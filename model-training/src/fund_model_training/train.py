from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import joblib
import pandas as pd

from .config import TrainingConfig, load_config
from .features import prepare_features
from .labels import ensure_label
from .metrics import build_report
from .onnx_export import export_lightgbm_to_onnx
from .schema import ID_TO_LABEL
from .split import time_series_split


def main() -> None:
    parser = argparse.ArgumentParser(description="Train fund prediction models.")
    parser.add_argument("--config", required=True, help="Path to a YAML config file.")
    args = parser.parse_args()
    cfg = load_config(args.config)
    train(cfg)


def train(cfg: TrainingConfig) -> None:
    try:
        from lightgbm import LGBMClassifier
    except ImportError as exc:
        raise SystemExit("Missing dependency: lightgbm. Run `pip install -r requirements.txt`.") from exc

    if not cfg.data_path.exists():
        raise FileNotFoundError(f"Training data not found: {cfg.data_path}")

    raw = pd.read_csv(cfg.data_path)
    labeled = ensure_label(
        raw,
        label_column=cfg.label_column,
        future_return_column=cfg.future_return_column,
        flat_threshold_pct=cfg.flat_threshold_pct,
    )
    samples, feature_names = prepare_features(labeled, cfg.feature_set)
    samples = samples.dropna(subset=[cfg.label_column])

    train_df, test_df = time_series_split(samples, cfg.test_size)
    x_train = train_df[feature_names].astype("float32")
    y_train = train_df[cfg.label_column].astype("int64")
    x_test = test_df[feature_names].astype("float32")
    y_test = test_df[cfg.label_column].astype("int64")

    params: dict[str, Any] = {
        "objective": "multiclass",
        "num_class": 3,
        "class_weight": "balanced",
        "random_state": cfg.random_seed,
        "n_jobs": -1,
        **cfg.lightgbm,
    }
    model = LGBMClassifier(**params)
    model.fit(x_train, y_train)

    predictions = model.predict(x_test)
    probabilities = model.predict_proba(x_test)
    report = build_report(y_test.to_numpy(), predictions, probabilities)

    cfg.report_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.report_output_path.write_text(
        json.dumps(report, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    export_lightgbm_to_onnx(model, len(feature_names), cfg.model_output_path)

    joblib_path = cfg.metadata_output_path.with_suffix(".joblib")
    joblib_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(model, joblib_path)

    metadata = {
        "task": cfg.task,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "feature_set": cfg.feature_set,
        "features": feature_names,
        "label_mapping": ID_TO_LABEL,
        "flat_threshold_pct": cfg.flat_threshold_pct,
        "train_rows": int(len(train_df)),
        "test_rows": int(len(test_df)),
        "model_output_path": str(cfg.model_output_path),
        "joblib_output_path": str(joblib_path),
        "report_output_path": str(cfg.report_output_path),
        "report_summary": {
            "accuracy": report["accuracy"],
            "balanced_accuracy": report["balanced_accuracy"],
            "high_confidence_coverage": report.get("high_confidence_coverage"),
            "high_confidence_accuracy": report.get("high_confidence_accuracy"),
        },
    }
    cfg.metadata_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.metadata_output_path.write_text(
        json.dumps(metadata, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    print(json.dumps(metadata["report_summary"], ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
