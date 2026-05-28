from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

import pandas as pd

from fund_model_training.features import prepare_features
from fund_model_training.labels import ensure_label
from fund_model_training.metrics import probability_calibration_report
from fund_model_training.onnx_export import default_classifier_onnx_path, export_classifier_sidecar
from fund_model_training.prediction_interval import empirical_prediction_interval_report
from fund_model_training.regime_evaluation import market_regime_report
from fund_model_training.return_decomposition import (
    ensure_return_decomposition_targets,
    fit_return_decomposition,
    predict_return,
    return_decomposition_metadata,
)
from fund_model_training.schema import ID_TO_LABEL
from fund_model_training.split import purged_time_series_split, time_series_split
from fund_model_training.train_baseline import _high_confidence_report, _labels_from_returns, _naive_baselines


MIN_CHAMPION_HIGH_CONFIDENCE_ACCURACY = 0.50
MIN_CHAMPION_HIGH_CONFIDENCE_COVERAGE = 0.05
MAX_CHAMPION_CALIBRATION_ECE = 0.12


@dataclass(frozen=True)
class TournamentConfig:
    task: str
    data_path: Path
    report_output_path: Path
    metadata_output_path: Path
    champion_output_path: Path
    classifier_onnx_output_path: Path | None = None
    feature_set: str = "index_fund_daily_v1"
    label_column: str = "label"
    future_return_column: str = "future_return_pct_next_day"
    regression_target: str = "future_return_pct_next_day"
    flat_threshold_pct: float = 0.05
    test_size: float = 0.2
    random_seed: int = 20260523
    high_confidence_threshold: float = 0.60
    candidates: tuple[str, ...] = (
        "lightgbm",
        "hist_gbdt",
        "random_forest",
        "extra_trees",
        "gradient_boosting",
        "logistic_ridge",
    )
    ablation_tests: tuple[str, ...] = ()
    purged_split: bool = True
    embargo_minutes: int | None = None
    require_classifier_onnx: bool = False
    rolling_backtest_folds: int = 3
    rolling_backtest_min_train_rows: int = 50
    rolling_backtest_min_test_rows: int = 5


def main() -> None:
    parser = argparse.ArgumentParser(description="Train multiple index-fund candidate models and select a champion.")
    parser.add_argument("--config", type=Path, help="YAML config path.")
    parser.add_argument("--data", type=Path, help="Processed sample CSV.")
    parser.add_argument("--report-output", type=Path, default=Path("reports/index_fund_tournament_report.json"))
    parser.add_argument("--metadata-output", type=Path, default=Path("artifacts/index_fund_tournament_metadata.json"))
    parser.add_argument("--champion-output", type=Path, default=Path("artifacts/index_fund_tournament_champion.joblib"))
    parser.add_argument("--feature-set", default="index_fund_daily_v1")
    parser.add_argument("--test-size", type=float, default=0.2)
    parser.add_argument("--high-confidence-threshold", type=float, default=0.60)
    args = parser.parse_args()

    cfg = load_tournament_config(args.config) if args.config else TournamentConfig(
        task="index_fund_model_tournament",
        data_path=_require_path(args.data, "--data is required when --config is not provided."),
        report_output_path=args.report_output,
        metadata_output_path=args.metadata_output,
        champion_output_path=args.champion_output,
        feature_set=args.feature_set,
        test_size=args.test_size,
        high_confidence_threshold=args.high_confidence_threshold,
    )
    metadata = train_tournament(cfg)
    print(json.dumps(metadata["champion_summary"], ensure_ascii=False, indent=2))


def load_tournament_config(path: str | Path) -> TournamentConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install -r requirements.txt`.") from exc

    config_path = Path(path)
    raw = yaml.safe_load(config_path.read_text(encoding="utf-8")) or {}
    base_dir = config_path.parent.parent

    def resolve(value: str) -> Path:
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    return TournamentConfig(
        task=str(raw.get("task", "index_fund_model_tournament")),
        data_path=resolve(str(raw["data_path"])),
        report_output_path=resolve(str(raw.get("report_output_path", "reports/index_fund_tournament_report.json"))),
        metadata_output_path=resolve(str(raw.get("metadata_output_path", "artifacts/index_fund_tournament_metadata.json"))),
        champion_output_path=resolve(str(raw.get("champion_output_path", "artifacts/index_fund_tournament_champion.joblib"))),
        classifier_onnx_output_path=resolve(str(raw["classifier_onnx_output_path"])) if raw.get("classifier_onnx_output_path") else None,
        feature_set=str(raw.get("feature_set", "index_fund_daily_v1")),
        label_column=str(raw.get("label_column", "label")),
        future_return_column=str(raw.get("future_return_column", "future_return_pct_next_day")),
        regression_target=str(raw.get("regression_target", raw.get("future_return_column", "future_return_pct_next_day"))),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.05)),
        test_size=float(raw.get("test_size", 0.2)),
        random_seed=int(raw.get("random_seed", 20260523)),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
        candidates=tuple(raw.get("candidates", TournamentConfig.__dataclass_fields__["candidates"].default)),
        ablation_tests=tuple(raw.get("ablation_tests", ())),
        purged_split=bool(raw.get("purged_split", True)),
        embargo_minutes=_optional_int(raw.get("embargo_minutes")),
        require_classifier_onnx=bool(raw.get("require_classifier_onnx", False)),
        rolling_backtest_folds=int(raw.get("rolling_backtest_folds", 3)),
        rolling_backtest_min_train_rows=int(raw.get("rolling_backtest_min_train_rows", 50)),
        rolling_backtest_min_test_rows=int(raw.get("rolling_backtest_min_test_rows", 5)),
    )


def train_tournament(cfg: TournamentConfig) -> dict[str, Any]:
    try:
        import joblib
    except ImportError as exc:
        raise SystemExit("Missing training dependencies. Run `pip install scikit-learn joblib PyYAML`.") from exc

    raw = pd.read_csv(cfg.data_path)
    labeled = ensure_label(raw, cfg.label_column, cfg.future_return_column, cfg.flat_threshold_pct)
    samples, feature_names = prepare_features(labeled, cfg.feature_set)
    samples, decomposition_target_cols = ensure_return_decomposition_targets(samples, cfg.regression_target)
    samples = samples.dropna(subset=[cfg.label_column, cfg.regression_target, "asof_time", *decomposition_target_cols]).copy()
    samples["asof_time"] = pd.to_datetime(samples["asof_time"], errors="coerce")
    samples = samples.dropna(subset=["asof_time"]).sort_values("asof_time").reset_index(drop=True)

    train_df, test_df, split_policy = _split_samples(samples, cfg)
    x_train = train_df[feature_names].astype("float32")
    y_train_cls = train_df[cfg.label_column].astype("int64")
    y_train_reg = pd.to_numeric(train_df[cfg.regression_target], errors="coerce").astype("float32")
    x_test = test_df[feature_names].astype("float32")
    y_test_cls = test_df[cfg.label_column].astype("int64")
    y_test_reg = pd.to_numeric(test_df[cfg.regression_target], errors="coerce").astype("float32")

    candidate_reports: list[dict[str, Any]] = []
    fitted: dict[str, dict[str, Any]] = {}
    for name in cfg.candidates:
        models = _candidate_models(name, cfg.random_seed)
        if models is None:
            candidate_reports.append({"candidate": name, "status": "skipped", "reason": _candidate_skip_reason(name)})
            continue
        cls_model, reg_model = models
        try:
            cls_model.fit(x_train, y_train_cls)
            reg_model.fit(x_train, y_train_reg)
            return_decomposition = fit_return_decomposition(reg_model, x_train, train_df, cfg.regression_target)
            report = {
                "candidate": name,
                "status": "ok",
                **_evaluate_fitted_models(
                    cls_model=cls_model,
                    reg_model=reg_model,
                    x_test=x_test,
                    y_test_cls=y_test_cls,
                    y_test_reg=y_test_reg,
                    test_df=test_df,
                    regression_target=cfg.regression_target,
                    return_decomposition=return_decomposition,
                    flat_threshold_pct=cfg.flat_threshold_pct,
                    high_confidence_threshold=cfg.high_confidence_threshold,
                ),
            }
            candidate_reports.append(report)
            fitted[name] = {
                "classifier": cls_model,
                "regressor": reg_model,
                "return_decomposition": return_decomposition,
            }
        except Exception as exc:
            candidate_reports.append({"candidate": name, "status": "failed", "reason": str(exc)})

    successful = [item for item in candidate_reports if item.get("status") == "ok"]
    if not successful:
        raise RuntimeError(f"No tournament candidate succeeded: {candidate_reports}")
    for item in successful:
        item["rolling_backtest"] = _run_rolling_backtest(
            samples=samples,
            feature_names=feature_names,
            candidate_name=str(item["candidate"]),
            cfg=cfg,
        )
    champion = sorted(successful, key=_champion_sort_key)[0]
    champion_name = str(champion["candidate"])
    ablations = _run_ablation_tests(
        names=cfg.ablation_tests,
        champion_name=champion_name,
        full_report=champion,
        feature_names=feature_names,
        train_df=train_df,
        test_df=test_df,
        label_column=cfg.label_column,
        regression_target=cfg.regression_target,
        random_seed=cfg.random_seed,
        flat_threshold_pct=cfg.flat_threshold_pct,
        high_confidence_threshold=cfg.high_confidence_threshold,
    )

    report = {
        "task": cfg.task,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "feature_set": cfg.feature_set,
        "features": feature_names,
        "return_decomposition": {
            "enabled": bool(decomposition_target_cols),
            "target_columns": decomposition_target_cols,
            "formula": "fund_return = tracking_index_return + tracking_error",
        },
        "label_mapping": ID_TO_LABEL,
        "candidates": candidate_reports,
        "champion": champion,
        "ablations": ablations,
        "naive_baselines": _naive_baselines(test_df, y_test_cls, y_test_reg, cfg.flat_threshold_pct),
        "walk_forward": {
            "train_rows": int(len(train_df)),
            "test_rows": int(len(test_df)),
            "train_start": str(train_df["asof_time"].min()),
            "train_end": str(train_df["asof_time"].max()),
            "test_start": str(test_df["asof_time"].min()),
            "test_end": str(test_df["asof_time"].max()),
            "sample_days": int(samples["asof_time"].dt.date.nunique()),
            "train_sample_days": int(train_df["asof_time"].dt.date.nunique()),
            "test_sample_days": int(test_df["asof_time"].dt.date.nunique()),
            "split_policy": split_policy,
        },
    }

    cfg.report_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.report_output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")

    champion_bundle = {
        **fitted[champion_name],
        "candidate": champion_name,
        "feature_names": feature_names,
        "config": asdict(cfg),
        "champion_report": champion,
        "prediction_interval": champion.get("prediction_interval"),
        "architecture": "tracking_index_plus_error" if fitted[champion_name].get("return_decomposition") else "direct_fund_return",
    }
    cfg.champion_output_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(champion_bundle, cfg.champion_output_path)
    classifier_onnx = export_classifier_sidecar(
        model=fitted[champion_name]["classifier"],
        feature_count=len(feature_names),
        output_path=cfg.classifier_onnx_output_path or default_classifier_onnx_path(cfg.champion_output_path),
        required=cfg.require_classifier_onnx,
    )

    metadata = {
        "task": cfg.task,
        "created_at": report["created_at"],
        "config": _jsonable(asdict(cfg)),
        "data_path": str(cfg.data_path),
        "feature_set": cfg.feature_set,
        "features": feature_names,
        "return_decomposition": return_decomposition_metadata(fitted[champion_name].get("return_decomposition")),
        "prediction_interval": champion.get("prediction_interval"),
        "champion": champion_name,
        "champion_output_path": str(cfg.champion_output_path),
        "classifier_onnx": classifier_onnx,
        "report_output_path": str(cfg.report_output_path),
        "train_rows": report["walk_forward"]["train_rows"],
        "test_rows": report["walk_forward"]["test_rows"],
        "walk_forward": report["walk_forward"],
        "champion_summary": champion,
        "ablations": ablations,
    }
    cfg.metadata_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.metadata_output_path.write_text(json.dumps(metadata, ensure_ascii=False, indent=2), encoding="utf-8")
    return metadata


def _candidate_models(name: str, random_seed: int):
    from sklearn.ensemble import (
        ExtraTreesClassifier,
        ExtraTreesRegressor,
        GradientBoostingClassifier,
        GradientBoostingRegressor,
        HistGradientBoostingClassifier,
        HistGradientBoostingRegressor,
        RandomForestClassifier,
        RandomForestRegressor,
    )
    from sklearn.linear_model import LogisticRegression, Ridge
    from sklearn.pipeline import make_pipeline
    from sklearn.preprocessing import StandardScaler

    if name == "hist_gbdt":
        return (
            HistGradientBoostingClassifier(learning_rate=0.05, max_iter=300, l2_regularization=0.05, random_state=random_seed),
            HistGradientBoostingRegressor(learning_rate=0.05, max_iter=300, l2_regularization=0.05, random_state=random_seed),
        )
    if name == "lightgbm":
        try:
            from lightgbm import LGBMClassifier, LGBMRegressor
        except ImportError:
            return None
        return (
            LGBMClassifier(
                n_estimators=600,
                learning_rate=0.03,
                num_leaves=31,
                min_child_samples=30,
                subsample=0.85,
                colsample_bytree=0.85,
                class_weight="balanced",
                random_state=random_seed,
                n_jobs=-1,
                verbosity=-1,
            ),
            LGBMRegressor(
                n_estimators=600,
                learning_rate=0.03,
                num_leaves=31,
                min_child_samples=30,
                subsample=0.85,
                colsample_bytree=0.85,
                random_state=random_seed,
                n_jobs=-1,
                verbosity=-1,
            ),
        )
    if name == "random_forest":
        return (
            RandomForestClassifier(n_estimators=350, min_samples_leaf=3, class_weight="balanced", random_state=random_seed, n_jobs=-1),
            RandomForestRegressor(n_estimators=350, min_samples_leaf=3, random_state=random_seed, n_jobs=-1),
        )
    if name == "extra_trees":
        return (
            ExtraTreesClassifier(n_estimators=450, min_samples_leaf=2, class_weight="balanced", random_state=random_seed, n_jobs=-1),
            ExtraTreesRegressor(n_estimators=450, min_samples_leaf=2, random_state=random_seed, n_jobs=-1),
        )
    if name == "gradient_boosting":
        return (
            GradientBoostingClassifier(n_estimators=250, learning_rate=0.04, max_depth=2, random_state=random_seed),
            GradientBoostingRegressor(n_estimators=250, learning_rate=0.04, max_depth=2, random_state=random_seed),
        )
    if name == "logistic_ridge":
        return (
            make_pipeline(StandardScaler(), LogisticRegression(max_iter=1000, class_weight="balanced", random_state=random_seed)),
            make_pipeline(StandardScaler(), Ridge(alpha=2.0, random_state=random_seed)),
        )
    return None


def _candidate_skip_reason(name: str) -> str:
    if name == "lightgbm":
        return "optional dependency lightgbm is not installed"
    return "unknown candidate"


def _split_samples(samples: pd.DataFrame, cfg: TournamentConfig) -> tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]:
    label_horizon = _label_horizon(cfg.regression_target)
    if not cfg.purged_split or label_horizon <= timedelta(0):
        train, test = time_series_split(samples, cfg.test_size)
        return train, test, {
            "type": "time_holdout",
            "test_size": cfg.test_size,
            "purged_split": False,
        }

    embargo = timedelta(minutes=cfg.embargo_minutes) if cfg.embargo_minutes is not None else label_horizon
    train, test, metadata = purged_time_series_split(
        samples,
        test_size=cfg.test_size,
        label_horizon=label_horizon,
        embargo=embargo,
    )
    metadata["test_size"] = cfg.test_size
    metadata["purged_split"] = True
    return train, test, metadata


def _label_horizon(regression_target: str) -> timedelta:
    if regression_target == "future_return_pct_3m":
        return timedelta(minutes=3)
    if regression_target == "future_return_pct_5m":
        return timedelta(minutes=5)
    return timedelta(0)


def _run_rolling_backtest(
    samples: pd.DataFrame,
    feature_names: list[str],
    candidate_name: str,
    cfg: TournamentConfig,
) -> dict[str, Any]:
    folds = _rolling_folds(samples, cfg)
    if not folds:
        return {
            "status": "skipped",
            "reason": (
                "not enough rows for rolling walk-forward backtest: "
                f"rows={len(samples)}, min_train_rows={cfg.rolling_backtest_min_train_rows}, "
                f"min_test_rows={cfg.rolling_backtest_min_test_rows}, folds={cfg.rolling_backtest_folds}"
            ),
        }

    reports: list[dict[str, Any]] = []
    for fold_index, (train_df, test_df, fold_meta) in enumerate(folds, start=1):
        models = _candidate_models(candidate_name, cfg.random_seed + fold_index)
        if models is None:
            return {"status": "skipped", "reason": _candidate_skip_reason(candidate_name)}
        cls_model, reg_model = models
        try:
            x_train = train_df[feature_names].astype("float32")
            y_train_cls = train_df[cfg.label_column].astype("int64")
            y_train_reg = pd.to_numeric(train_df[cfg.regression_target], errors="coerce").astype("float32")
            x_test = test_df[feature_names].astype("float32")
            y_test_cls = test_df[cfg.label_column].astype("int64")
            y_test_reg = pd.to_numeric(test_df[cfg.regression_target], errors="coerce").astype("float32")
            cls_model.fit(x_train, y_train_cls)
            reg_model.fit(x_train, y_train_reg)
            return_decomposition = fit_return_decomposition(reg_model, x_train, train_df, cfg.regression_target)
            metrics = _evaluate_fitted_models(
                cls_model=cls_model,
                reg_model=reg_model,
                x_test=x_test,
                y_test_cls=y_test_cls,
                y_test_reg=y_test_reg,
                test_df=test_df,
                regression_target=cfg.regression_target,
                return_decomposition=return_decomposition,
                flat_threshold_pct=cfg.flat_threshold_pct,
                high_confidence_threshold=cfg.high_confidence_threshold,
            )
            reports.append({
                "fold": fold_index,
                **fold_meta,
                **metrics,
            })
        except Exception as exc:
            reports.append({
                "fold": fold_index,
                **fold_meta,
                "status": "failed",
                "reason": str(exc),
            })

    ok_reports = [item for item in reports if item.get("status") != "failed"]
    if not ok_reports:
        return {"status": "failed", "folds": reports, "reason": "all rolling folds failed"}
    return {
        "status": "ok",
        "folds": reports,
        "summary": _rolling_summary(ok_reports),
    }


def _rolling_folds(samples: pd.DataFrame, cfg: TournamentConfig) -> list[tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]]:
    if cfg.rolling_backtest_folds <= 0:
        return []
    ordered = samples.sort_values("asof_time").reset_index(drop=True)
    n_rows = len(ordered)
    max_test_rows = n_rows - cfg.rolling_backtest_min_train_rows
    if max_test_rows < cfg.rolling_backtest_min_test_rows:
        return []

    requested_folds = max(int(cfg.rolling_backtest_folds), 1)
    total_test_rows = min(max_test_rows, max(requested_folds * cfg.rolling_backtest_min_test_rows, int(n_rows * cfg.test_size)))
    fold_test_rows = max(cfg.rolling_backtest_min_test_rows, total_test_rows // requested_folds)
    fold_count = min(requested_folds, max_test_rows // fold_test_rows)
    if fold_count <= 0:
        return []

    start = n_rows - fold_count * fold_test_rows
    label_horizon = _label_horizon(cfg.regression_target)
    embargo = timedelta(minutes=cfg.embargo_minutes) if cfg.embargo_minutes is not None else label_horizon
    folds: list[tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]] = []
    for fold_idx in range(fold_count):
        test_start_idx = start + fold_idx * fold_test_rows
        test_end_idx = test_start_idx + fold_test_rows
        train_df = ordered.iloc[:test_start_idx].copy()
        test_df = ordered.iloc[test_start_idx:test_end_idx].copy()
        if len(train_df) < cfg.rolling_backtest_min_train_rows or len(test_df) < cfg.rolling_backtest_min_test_rows:
            continue

        original_train_rows = len(train_df)
        purge_metadata: dict[str, Any] = {}
        if cfg.purged_split and label_horizon > timedelta(0):
            test_start_time = pd.to_datetime(test_df["asof_time"], errors="coerce").min()
            cutoff = test_start_time - embargo
            keep = (pd.to_datetime(train_df["asof_time"], errors="coerce") + label_horizon) < cutoff
            train_df = train_df.loc[keep].copy()
            purge_metadata = {
                "purged_split": True,
                "label_horizon_seconds": int(label_horizon.total_seconds()),
                "embargo_seconds": int(embargo.total_seconds()),
                "original_train_rows": int(original_train_rows),
                "purged_train_rows": int(len(train_df)),
                "removed_train_rows": int(original_train_rows - len(train_df)),
            }
        else:
            purge_metadata = {"purged_split": False}
        if len(train_df) < cfg.rolling_backtest_min_train_rows:
            continue

        metadata = {
            "train_rows": int(len(train_df)),
            "test_rows": int(len(test_df)),
            "train_start": str(train_df["asof_time"].min()),
            "train_end": str(train_df["asof_time"].max()),
            "test_start": str(test_df["asof_time"].min()),
            "test_end": str(test_df["asof_time"].max()),
            **purge_metadata,
        }
        folds.append((train_df, test_df, metadata))
    return folds


def _rolling_summary(fold_reports: list[dict[str, Any]]) -> dict[str, Any]:
    return {
        "fold_count": int(len(fold_reports)),
        "mean_classification_accuracy": _mean_metric(fold_reports, "classification_accuracy"),
        "mean_balanced_accuracy": _mean_metric(fold_reports, "balanced_accuracy"),
        "mean_regression_mae": _mean_metric(fold_reports, "regression_mae"),
        "mean_regression_rmse": _mean_metric(fold_reports, "regression_rmse"),
        "mean_direction_accuracy_from_regression": _mean_metric(fold_reports, "direction_accuracy_from_regression"),
        "mean_high_confidence_coverage": _mean_nested_metric(fold_reports, "high_confidence", "coverage"),
        "mean_high_confidence_accuracy": _mean_nested_metric(fold_reports, "high_confidence", "accuracy"),
        "weak_slice_count": int(sum(len((item.get("market_regime") or {}).get("weak_slices") or []) for item in fold_reports)),
    }


def _mean_metric(items: list[dict[str, Any]], key: str) -> float | None:
    values = [_float(item.get(key)) for item in items]
    values = [value for value in values if value is not None]
    if not values:
        return None
    return round(sum(values) / len(values), 8)


def _mean_nested_metric(items: list[dict[str, Any]], parent: str, key: str) -> float | None:
    values = [_float((item.get(parent) or {}).get(key)) for item in items]
    values = [value for value in values if value is not None]
    if not values:
        return None
    return round(sum(values) / len(values), 8)


def _evaluate_fitted_models(
    cls_model,
    reg_model,
    x_test,
    y_test_cls,
    y_test_reg,
    test_df,
    regression_target: str,
    return_decomposition: dict[str, Any] | None,
    flat_threshold_pct: float,
    high_confidence_threshold: float,
) -> dict[str, Any]:
    from sklearn.metrics import accuracy_score, balanced_accuracy_score, mean_absolute_error, mean_squared_error, r2_score

    cls_pred = cls_model.predict(x_test)
    return_prediction = predict_return(reg_model, x_test, return_decomposition)
    reg_pred = return_prediction["prediction"]
    probabilities = cls_model.predict_proba(x_test) if hasattr(cls_model, "predict_proba") else None
    component_metrics = _return_component_metrics(return_prediction, test_df, regression_target)
    regime_report = market_regime_report(
        test_df=test_df,
        y_true_cls=y_test_cls.to_numpy(),
        y_pred_cls=cls_pred,
        y_true_reg=y_test_reg.to_numpy(),
        y_pred_reg=reg_pred,
        probabilities=probabilities,
        high_confidence_threshold=high_confidence_threshold,
    )
    return {
        "classification_accuracy": float(accuracy_score(y_test_cls, cls_pred)),
        "balanced_accuracy": float(balanced_accuracy_score(y_test_cls, cls_pred)),
        "regression_mae": float(mean_absolute_error(y_test_reg, reg_pred)),
        "regression_rmse": float(mean_squared_error(y_test_reg, reg_pred) ** 0.5),
        "regression_r2": float(r2_score(y_test_reg, reg_pred)) if len(y_test_reg) > 1 else None,
        "direction_accuracy_from_regression": float(accuracy_score(
            y_test_cls,
            _labels_from_returns(reg_pred, flat_threshold_pct),
        )),
        "high_confidence": _high_confidence_report(
            y_test_cls.to_numpy(),
            cls_pred,
            probabilities,
            high_confidence_threshold,
        ),
        "calibration": probability_calibration_report(y_test_cls.to_numpy(), cls_pred, probabilities),
        "prediction_interval": empirical_prediction_interval_report(y_test_reg.to_numpy(), reg_pred),
        "return_decomposition": {
            **return_decomposition_metadata(return_decomposition),
            **component_metrics,
        },
        "market_regime": regime_report,
    }


def _run_ablation_tests(
    names: tuple[str, ...],
    champion_name: str,
    full_report: dict[str, Any],
    feature_names: list[str],
    train_df,
    test_df,
    label_column: str,
    regression_target: str,
    random_seed: int,
    flat_threshold_pct: float,
    high_confidence_threshold: float,
) -> list[dict[str, Any]]:
    reports = []
    seen: set[str] = set()
    for raw_name in names:
        name = str(raw_name).strip()
        if not name or name in seen:
            continue
        seen.add(name)
        selected = _ablation_feature_names(feature_names, name)
        removed = [feature for feature in feature_names if feature not in selected]
        if name != "full_feature_model" and not removed:
            reports.append({
                "ablation": name,
                "status": "skipped",
                "reason": "no matching features were present in this feature set",
                "candidate": champion_name,
                "removed_features": [],
                "feature_count": len(feature_names),
            })
            continue
        if not selected:
            reports.append({
                "ablation": name,
                "status": "skipped",
                "reason": "ablation removed every feature",
                "candidate": champion_name,
                "removed_features": removed,
                "feature_count": 0,
            })
            continue
        if name == "full_feature_model":
            reports.append({
                "ablation": name,
                "status": "ok",
                "candidate": champion_name,
                "removed_features": [],
                "feature_count": len(feature_names),
                **_ablation_metric_block(full_report, full_report),
            })
            continue

        models = _candidate_models(champion_name, random_seed)
        if models is None:
            reports.append({
                "ablation": name,
                "status": "skipped",
                "reason": f"unknown champion candidate: {champion_name}",
                "candidate": champion_name,
                "removed_features": removed,
                "feature_count": len(selected),
            })
            continue
        cls_model, reg_model = models
        try:
            x_train = train_df[selected].astype("float32")
            y_train_cls = train_df[label_column].astype("int64")
            y_train_reg = pd.to_numeric(train_df[regression_target], errors="coerce").astype("float32")
            x_test = test_df[selected].astype("float32")
            y_test_cls = test_df[label_column].astype("int64")
            y_test_reg = pd.to_numeric(test_df[regression_target], errors="coerce").astype("float32")
            cls_model.fit(x_train, y_train_cls)
            reg_model.fit(x_train, y_train_reg)
            return_decomposition = fit_return_decomposition(reg_model, x_train, train_df, regression_target)
            metrics = _evaluate_fitted_models(
                cls_model=cls_model,
                reg_model=reg_model,
                x_test=x_test,
                y_test_cls=y_test_cls,
                y_test_reg=y_test_reg,
                test_df=test_df,
                regression_target=regression_target,
                return_decomposition=return_decomposition,
                flat_threshold_pct=flat_threshold_pct,
                high_confidence_threshold=high_confidence_threshold,
            )
            reports.append({
                "ablation": name,
                "status": "ok",
                "candidate": champion_name,
                "removed_features": removed,
                "feature_count": len(selected),
                **_ablation_metric_block(metrics, full_report),
            })
        except Exception as exc:
            reports.append({
                "ablation": name,
                "status": "failed",
                "reason": str(exc),
                "candidate": champion_name,
                "removed_features": removed,
                "feature_count": len(selected),
            })
    return reports


def _return_component_metrics(return_prediction: dict[str, Any], test_df, regression_target: str) -> dict[str, Any]:
    from sklearn.metrics import mean_absolute_error

    if return_prediction.get("method") != "tracking_index_plus_error":
        return {}
    target_columns = _component_target_columns(regression_target)
    if target_columns is None:
        return {}
    index_col, tracking_error_col = target_columns
    if index_col not in test_df.columns or tracking_error_col not in test_df.columns:
        return {}

    metrics: dict[str, Any] = {}
    index_pred = return_prediction.get("index_return")
    tracking_error_pred = return_prediction.get("tracking_error")
    direct_pred = return_prediction.get("direct_fund_return")
    if index_pred is not None:
        metrics["index_return_mae"] = float(mean_absolute_error(pd.to_numeric(test_df[index_col], errors="coerce"), index_pred))
    if tracking_error_pred is not None:
        metrics["tracking_error_mae"] = float(mean_absolute_error(pd.to_numeric(test_df[tracking_error_col], errors="coerce"), tracking_error_pred))
    if direct_pred is not None:
        metrics["direct_fund_return_mae"] = float(mean_absolute_error(pd.to_numeric(test_df[regression_target], errors="coerce"), direct_pred))
    return metrics


def _component_target_columns(regression_target: str) -> tuple[str, str] | None:
    if regression_target == "future_return_pct_next_day":
        return "future_index_return_pct_next_day", "future_tracking_error_pct_next_day"
    if regression_target == "future_return_pct_1w":
        return "future_index_return_pct_1w", "future_tracking_error_pct_1w"
    if regression_target == "future_return_pct_3m":
        return "future_index_return_pct_3m", "future_tracking_error_pct_3m"
    if regression_target == "future_return_pct_5m":
        return "future_index_return_pct_5m", "future_tracking_error_pct_5m"
    return None


def _ablation_feature_names(feature_names: list[str], ablation: str) -> list[str]:
    if ablation == "full_feature_model":
        return list(feature_names)
    rules = {
        "without_panic_factor": lambda feature: (
            feature == "fear_score" or
            feature.startswith("panic_") or
            "panic" in feature
        ),
        "without_futures_commodity": lambda feature: (
            feature.startswith("futures_") or
            feature.startswith("commodity_") or
            feature in {"futures_basis", "futures_open_interest_change_5d"}
        ),
        "without_sentiment": lambda feature: any(
            token in feature
            for token in ("sentiment", "news", "topic", "entity", "policy", "geopolitical")
        ),
        "without_cross_market": lambda feature: (
            feature.startswith("cross_market_") or
            feature.startswith("global_") or
            feature.startswith("fx_") or
            feature.startswith("rate_") or
            feature.startswith("market_")
        ),
        "without_index_features": lambda feature: feature.startswith("index_") or feature.startswith("fund_index_"),
    }
    should_remove = rules.get(ablation)
    if should_remove is None:
        return list(feature_names)
    return [feature for feature in feature_names if not should_remove(feature)]


def _ablation_metric_block(metrics: dict[str, Any], full_report: dict[str, Any]) -> dict[str, Any]:
    high_confidence = metrics.get("high_confidence") or {}
    full_high_confidence = full_report.get("high_confidence") or {}
    return {
        "classification_accuracy": metrics.get("classification_accuracy"),
        "balanced_accuracy": metrics.get("balanced_accuracy"),
        "regression_mae": metrics.get("regression_mae"),
        "regression_rmse": metrics.get("regression_rmse"),
        "direction_accuracy_from_regression": metrics.get("direction_accuracy_from_regression"),
        "high_confidence": high_confidence,
        "calibration": metrics.get("calibration"),
        "return_decomposition": metrics.get("return_decomposition"),
        "market_regime": metrics.get("market_regime"),
        "delta_vs_full": {
            "classification_accuracy": _metric_delta(metrics.get("classification_accuracy"), full_report.get("classification_accuracy")),
            "balanced_accuracy": _metric_delta(metrics.get("balanced_accuracy"), full_report.get("balanced_accuracy")),
            "regression_mae": _metric_delta(metrics.get("regression_mae"), full_report.get("regression_mae")),
            "high_confidence_coverage": _metric_delta(high_confidence.get("coverage"), full_high_confidence.get("coverage")),
            "high_confidence_accuracy": _metric_delta(high_confidence.get("accuracy"), full_high_confidence.get("accuracy")),
            "calibration_ece": _metric_delta(
                (metrics.get("calibration") or {}).get("ece"),
                (full_report.get("calibration") or {}).get("ece"),
            ),
        },
    }


def _metric_delta(value: Any, baseline: Any) -> float | None:
    if value is None or baseline is None:
        return None
    try:
        return round(float(value) - float(baseline), 8)
    except (TypeError, ValueError):
        return None


def _float(value: Any) -> float | None:
    try:
        if value is None:
            return None
        return float(value)
    except (TypeError, ValueError):
        return None


def _champion_sort_key(report: dict[str, Any]):
    high_confidence = report.get("high_confidence", {})
    high_conf_acc = high_confidence.get("accuracy")
    if high_conf_acc is None:
        high_conf_acc = -1.0
    high_conf_coverage = high_confidence.get("coverage")
    if high_conf_coverage is None:
        high_conf_coverage = 0.0
    calibration_ece = report.get("calibration", {}).get("ece")
    if calibration_ece is None:
        calibration_ece = 1.0
    has_actionable_slice = _has_actionable_high_confidence_slice(report)
    actionable_high_conf_acc = high_conf_acc if has_actionable_slice else -1.0
    actionable_high_conf_coverage = high_conf_coverage if has_actionable_slice else 0.0
    rolling_summary = (report.get("rolling_backtest") or {}).get("summary") or {}
    rolling_balanced_accuracy = rolling_summary.get("mean_balanced_accuracy")
    if rolling_balanced_accuracy is None:
        rolling_balanced_accuracy = report.get("balanced_accuracy", 0.0)
    rolling_mae = rolling_summary.get("mean_regression_mae")
    if rolling_mae is None:
        rolling_mae = report.get("regression_mae", 0.0)
    return (
        -float(has_actionable_slice),
        -float(actionable_high_conf_acc),
        -float(actionable_high_conf_coverage),
        -float(rolling_balanced_accuracy),
        float(rolling_mae),
        -float(report["balanced_accuracy"]),
        -float(report["classification_accuracy"]),
        float(report["regression_mae"]),
        float(calibration_ece),
        -float(high_conf_acc),
    )


def _has_actionable_high_confidence_slice(report: dict[str, Any]) -> bool:
    high_confidence = report.get("high_confidence") or {}
    high_conf_acc = _float(high_confidence.get("accuracy"))
    high_conf_coverage = _float(high_confidence.get("coverage"))
    calibration_ece = _float((report.get("calibration") or {}).get("ece"))
    if high_conf_acc is None or high_conf_coverage is None or calibration_ece is None:
        return False
    return (
        high_conf_acc >= MIN_CHAMPION_HIGH_CONFIDENCE_ACCURACY and
        high_conf_coverage >= MIN_CHAMPION_HIGH_CONFIDENCE_COVERAGE and
        calibration_ece <= MAX_CHAMPION_CALIBRATION_ECE
    )


def _require_path(value: Path | None, message: str) -> Path:
    if value is None:
        raise SystemExit(message)
    return value


def _optional_int(value: Any) -> int | None:
    if value is None or value == "":
        return None
    return int(value)


def _jsonable(value: Any) -> Any:
    if isinstance(value, Path):
        return str(value)
    if isinstance(value, tuple):
        return [_jsonable(item) for item in value]
    if isinstance(value, list):
        return [_jsonable(item) for item in value]
    if isinstance(value, dict):
        return {key: _jsonable(item) for key, item in value.items()}
    return value


if __name__ == "__main__":
    main()
