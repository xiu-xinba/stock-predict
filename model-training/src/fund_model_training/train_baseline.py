from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd

from fund_model_training.features import prepare_features
from fund_model_training.labels import ensure_label
from fund_model_training.schema import ID_TO_LABEL


@dataclass(frozen=True)
class BaselineConfig:
    task: str
    data_path: Path
    report_output_path: Path
    metadata_output_path: Path
    model_output_path: Path
    feature_set: str = "index_fund_daily_v1"
    label_column: str = "label"
    future_return_column: str = "future_return_pct_next_day"
    regression_target: str = "future_return_pct_next_day"
    flat_threshold_pct: float = 0.05
    test_size: float = 0.2
    random_seed: int = 20260523
    high_confidence_threshold: float = 0.60


def main() -> None:
    parser = argparse.ArgumentParser(description="Train a walk-forward index-fund baseline model.")
    parser.add_argument("--config", type=Path, help="YAML config path.")
    parser.add_argument("--data", type=Path, help="Processed sample CSV.")
    parser.add_argument("--feature-set", default="index_fund_daily_v1")
    parser.add_argument("--report-output", type=Path, default=Path("reports/index_fund_baseline_report.json"))
    parser.add_argument("--metadata-output", type=Path, default=Path("artifacts/index_fund_baseline_metadata.json"))
    parser.add_argument("--model-output", type=Path, default=Path("artifacts/index_fund_baseline.joblib"))
    parser.add_argument("--test-size", type=float, default=0.2)
    parser.add_argument("--flat-threshold-pct", type=float, default=0.05)
    parser.add_argument("--high-confidence-threshold", type=float, default=0.60)
    args = parser.parse_args()

    cfg = load_baseline_config(args.config) if args.config else BaselineConfig(
        task="index_fund_daily_baseline",
        data_path=_require_path(args.data, "--data is required when --config is not provided."),
        report_output_path=args.report_output,
        metadata_output_path=args.metadata_output,
        model_output_path=args.model_output,
        feature_set=args.feature_set,
        test_size=args.test_size,
        flat_threshold_pct=args.flat_threshold_pct,
        high_confidence_threshold=args.high_confidence_threshold,
    )
    metadata = train_baseline(cfg)
    print(json.dumps(metadata["report_summary"], ensure_ascii=False, indent=2))


def load_baseline_config(path: str | Path) -> BaselineConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install -r requirements.txt`.") from exc

    config_path = Path(path)
    with config_path.open("r", encoding="utf-8") as fh:
        raw = yaml.safe_load(fh) or {}
    base_dir = config_path.parent.parent

    def resolve(value: str) -> Path:
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    return BaselineConfig(
        task=str(raw.get("task", "index_fund_daily_baseline")),
        data_path=resolve(str(raw["data_path"])),
        report_output_path=resolve(str(raw.get("report_output_path", "reports/index_fund_baseline_report.json"))),
        metadata_output_path=resolve(str(raw.get("metadata_output_path", "artifacts/index_fund_baseline_metadata.json"))),
        model_output_path=resolve(str(raw.get("model_output_path", "artifacts/index_fund_baseline.joblib"))),
        feature_set=str(raw.get("feature_set", "index_fund_daily_v1")),
        label_column=str(raw.get("label_column", "label")),
        future_return_column=str(raw.get("future_return_column", "future_return_pct_next_day")),
        regression_target=str(raw.get("regression_target", raw.get("future_return_column", "future_return_pct_next_day"))),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.05)),
        test_size=float(raw.get("test_size", 0.2)),
        random_seed=int(raw.get("random_seed", 20260523)),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
    )


def train_baseline(cfg: BaselineConfig) -> dict[str, Any]:
    try:
        import joblib
        from sklearn.ensemble import HistGradientBoostingClassifier, HistGradientBoostingRegressor, RandomForestClassifier
        from sklearn.inspection import permutation_importance
        from sklearn.metrics import (
            accuracy_score,
            balanced_accuracy_score,
            confusion_matrix,
            mean_absolute_error,
            mean_squared_error,
            r2_score,
        )
    except ImportError as exc:
        raise SystemExit("Missing training dependencies. Run `pip install scikit-learn joblib PyYAML`.") from exc

    if not cfg.data_path.exists():
        raise FileNotFoundError(f"Training data not found: {cfg.data_path}")

    raw = pd.read_csv(cfg.data_path)
    labeled = ensure_label(raw, cfg.label_column, cfg.future_return_column, cfg.flat_threshold_pct)
    samples, feature_names = prepare_features(labeled, cfg.feature_set)
    samples = samples.dropna(subset=[cfg.label_column, cfg.regression_target, "asof_time"]).copy()
    samples["asof_time"] = pd.to_datetime(samples["asof_time"], errors="coerce")
    samples = samples.dropna(subset=["asof_time"]).sort_values("asof_time").reset_index(drop=True)

    train_df, test_df = _time_split(samples, cfg.test_size)
    x_train = train_df[feature_names].astype("float32")
    y_train_cls = train_df[cfg.label_column].astype("int64")
    y_train_reg = pd.to_numeric(train_df[cfg.regression_target], errors="coerce").astype("float32")
    x_test = test_df[feature_names].astype("float32")
    y_test_cls = test_df[cfg.label_column].astype("int64")
    y_test_reg = pd.to_numeric(test_df[cfg.regression_target], errors="coerce").astype("float32")

    cls_model = _make_classifier(cfg.random_seed)
    reg_model = HistGradientBoostingRegressor(
        learning_rate=0.05,
        max_iter=250,
        l2_regularization=0.05,
        random_state=cfg.random_seed,
    )
    cls_model.fit(x_train, y_train_cls)
    reg_model.fit(x_train, y_train_reg)

    cls_pred = cls_model.predict(x_test)
    reg_pred = reg_model.predict(x_test)
    probabilities = _predict_proba(cls_model, x_test)

    report = {
        "task": cfg.task,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "classification": {
            "accuracy": float(accuracy_score(y_test_cls, cls_pred)),
            "balanced_accuracy": float(balanced_accuracy_score(y_test_cls, cls_pred)),
            "confusion_matrix": confusion_matrix(y_test_cls, cls_pred, labels=[0, 1, 2]).tolist(),
            "label_mapping": ID_TO_LABEL,
        },
        "regression": {
            "target": cfg.regression_target,
            "mae": float(mean_absolute_error(y_test_reg, reg_pred)),
            "rmse": float(mean_squared_error(y_test_reg, reg_pred) ** 0.5),
            "r2": float(r2_score(y_test_reg, reg_pred)) if len(y_test_reg) > 1 else None,
            "direction_accuracy_from_regression": float(accuracy_score(
                y_test_cls,
                _labels_from_returns(reg_pred, cfg.flat_threshold_pct),
            )),
        },
        "naive_baselines": _naive_baselines(test_df, y_test_cls, y_test_reg, cfg.flat_threshold_pct),
        "high_confidence": _high_confidence_report(y_test_cls.to_numpy(), cls_pred, probabilities, cfg.high_confidence_threshold),
        "feature_importance": _feature_importance(cls_model, x_test, y_test_cls, feature_names, permutation_importance),
        "walk_forward": {
            "train_rows": int(len(train_df)),
            "test_rows": int(len(test_df)),
            "train_start": str(train_df["asof_time"].min()),
            "train_end": str(train_df["asof_time"].max()),
            "test_start": str(test_df["asof_time"].min()),
            "test_end": str(test_df["asof_time"].max()),
        },
    }

    cfg.report_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.report_output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")

    model_bundle = {
        "classifier": cls_model,
        "regressor": reg_model,
        "feature_names": feature_names,
        "config": asdict(cfg),
    }
    cfg.model_output_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(model_bundle, cfg.model_output_path)

    metadata = {
        "task": cfg.task,
        "created_at": report["created_at"],
        "feature_set": cfg.feature_set,
        "features": feature_names,
        "model_output_path": str(cfg.model_output_path),
        "report_output_path": str(cfg.report_output_path),
        "train_rows": report["walk_forward"]["train_rows"],
        "test_rows": report["walk_forward"]["test_rows"],
        "report_summary": {
            "classification_accuracy": report["classification"]["accuracy"],
            "balanced_accuracy": report["classification"]["balanced_accuracy"],
            "regression_mae": report["regression"]["mae"],
            "regression_rmse": report["regression"]["rmse"],
            "high_confidence_coverage": report["high_confidence"]["coverage"],
            "high_confidence_accuracy": report["high_confidence"]["accuracy"],
        },
    }
    cfg.metadata_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.metadata_output_path.write_text(json.dumps(metadata, ensure_ascii=False, indent=2), encoding="utf-8")
    return metadata


def _make_classifier(random_seed: int):
    from sklearn.ensemble import HistGradientBoostingClassifier

    return HistGradientBoostingClassifier(
        learning_rate=0.05,
        max_iter=250,
        l2_regularization=0.05,
        random_state=random_seed,
    )


def _predict_proba(model, x_test):
    if hasattr(model, "predict_proba"):
        return model.predict_proba(x_test)
    return None


def _time_split(samples: pd.DataFrame, test_size: float) -> tuple[pd.DataFrame, pd.DataFrame]:
    if not 0 < test_size < 1:
        raise ValueError("test_size must be between 0 and 1.")
    split_idx = int(len(samples) * (1.0 - test_size))
    if split_idx <= 0 or split_idx >= len(samples):
        raise ValueError("Not enough rows for a train/test split.")
    return samples.iloc[:split_idx].copy(), samples.iloc[split_idx:].copy()


def _labels_from_returns(values, flat_threshold_pct: float):
    labels = np.ones(len(values), dtype="int64")
    values = np.asarray(values)
    labels[values < -flat_threshold_pct] = 0
    labels[values > flat_threshold_pct] = 2
    return labels


def _high_confidence_report(y_true, y_pred, probabilities, threshold: float) -> dict[str, Any]:
    from sklearn.metrics import accuracy_score

    if probabilities is None or len(probabilities) == 0:
        return {"threshold": threshold, "coverage": 0.0, "accuracy": None}
    max_prob = np.max(probabilities, axis=1)
    mask = max_prob >= threshold
    if not np.any(mask):
        return {"threshold": threshold, "coverage": 0.0, "accuracy": None}
    return {
        "threshold": threshold,
        "coverage": float(np.mean(mask)),
        "accuracy": float(accuracy_score(np.asarray(y_true)[mask], np.asarray(y_pred)[mask])),
    }


def _naive_baselines(test_df, y_test_cls, y_test_reg, flat_threshold_pct: float) -> dict[str, Any]:
    from sklearn.metrics import accuracy_score, mean_absolute_error, mean_squared_error

    prev_return = pd.to_numeric(test_df.get("fund_return_1d", 0.0), errors="coerce").fillna(0.0).to_numpy()
    index_return = pd.to_numeric(test_df.get("index_return_1d", 0.0), errors="coerce").fillna(0.0).to_numpy()
    zero_return = np.zeros(len(test_df))
    return {
        "previous_fund_return": _baseline_metrics(y_test_cls, y_test_reg, prev_return, flat_threshold_pct, accuracy_score, mean_absolute_error, mean_squared_error),
        "same_day_index_return": _baseline_metrics(y_test_cls, y_test_reg, index_return, flat_threshold_pct, accuracy_score, mean_absolute_error, mean_squared_error),
        "zero_return": _baseline_metrics(y_test_cls, y_test_reg, zero_return, flat_threshold_pct, accuracy_score, mean_absolute_error, mean_squared_error),
    }


def _baseline_metrics(y_cls, y_reg, pred_return, flat_threshold_pct, accuracy_score, mean_absolute_error, mean_squared_error):
    return {
        "direction_accuracy": float(accuracy_score(y_cls, _labels_from_returns(pred_return, flat_threshold_pct))),
        "mae": float(mean_absolute_error(y_reg, pred_return)),
        "rmse": float(mean_squared_error(y_reg, pred_return) ** 0.5),
    }


def _feature_importance(model, x_test, y_test, feature_names: list[str], permutation_importance) -> list[dict[str, Any]]:
    try:
        result = permutation_importance(model, x_test, y_test, n_repeats=5, random_state=20260523)
    except Exception:
        return []
    order = np.argsort(result.importances_mean)[::-1][:20]
    return [
        {
            "feature": feature_names[int(idx)],
            "importance_mean": float(result.importances_mean[int(idx)]),
            "importance_std": float(result.importances_std[int(idx)]),
        }
        for idx in order
    ]


def _require_path(value: Path | None, message: str) -> Path:
    if value is None:
        raise SystemExit(message)
    return value


if __name__ == "__main__":
    main()
