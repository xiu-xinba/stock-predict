from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from fund_model_training.drift_report import build_drift_report
from fund_model_training.prediction_logging import backfill_prediction_log, evaluate_prediction_log


@dataclass(frozen=True)
class MonitoringCycleConfig:
    prediction_log_path: Path
    samples_path: Path
    feature_set: str
    backfilled_log_output_path: Path
    performance_report_output_path: Path
    drift_report_output_path: Path
    summary_output_path: Path
    flat_threshold_pct: float = 0.02
    high_confidence_threshold: float = 0.60
    round_trip_cost_pct: float = 0.0
    reference_rows: int = 1000
    current_rows: int = 250
    psi_bins: int = 10
    psi_threshold: float = 0.20
    ks_threshold: float = 0.20
    missing_delta_threshold: float = 0.10
    skip_missing_prediction_log: bool = True
    horizons: tuple[str, ...] = ()


def main() -> None:
    parser = argparse.ArgumentParser(description="Run prediction backfill, performance evaluation, and drift reporting.")
    parser.add_argument("--config", type=Path, required=True, help="Monitoring cycle YAML config.")
    args = parser.parse_args()

    cfg = load_monitoring_cycle_config(args.config)
    summary = run_monitoring_cycle(cfg)
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def load_monitoring_cycle_config(path: str | Path) -> MonitoringCycleConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install -r requirements.txt`.") from exc

    config_path = Path(path)
    raw = yaml.safe_load(config_path.read_text(encoding="utf-8")) or {}
    base_dir = config_path.parent.parent

    def resolve(value: str | Path) -> Path:
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    drift = raw.get("drift") or {}
    return MonitoringCycleConfig(
        prediction_log_path=resolve(raw.get("prediction_log_path", "data/processed/prediction_log.csv")),
        samples_path=resolve(raw["samples_path"]),
        feature_set=str(raw["feature_set"]),
        backfilled_log_output_path=resolve(raw.get("backfilled_log_output_path", "data/processed/prediction_log_backfilled.csv")),
        performance_report_output_path=resolve(raw.get("performance_report_output_path", "reports/prediction_performance_report.json")),
        drift_report_output_path=resolve(raw.get("drift_report_output_path", "reports/feature_drift_report.json")),
        summary_output_path=resolve(raw.get("summary_output_path", "reports/monitoring_cycle_summary.json")),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.02)),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
        round_trip_cost_pct=float(raw.get("round_trip_cost_pct", 0.0)),
        reference_rows=int(drift.get("reference_rows", raw.get("reference_rows", 1000))),
        current_rows=int(drift.get("current_rows", raw.get("current_rows", 250))),
        psi_bins=int(drift.get("psi_bins", raw.get("psi_bins", 10))),
        psi_threshold=float(drift.get("psi_threshold", raw.get("psi_threshold", 0.20))),
        ks_threshold=float(drift.get("ks_threshold", raw.get("ks_threshold", 0.20))),
        missing_delta_threshold=float(drift.get("missing_delta_threshold", raw.get("missing_delta_threshold", 0.10))),
        skip_missing_prediction_log=bool(raw.get("skip_missing_prediction_log", True)),
        horizons=tuple(str(item) for item in raw.get("horizons", [])),
    )


def run_monitoring_cycle(cfg: MonitoringCycleConfig) -> dict[str, Any]:
    prediction_block = _run_prediction_performance(cfg)
    drift_report = build_drift_report(
        samples_path=cfg.samples_path,
        feature_set=cfg.feature_set,
        output_path=cfg.drift_report_output_path,
        reference_rows=cfg.reference_rows,
        current_rows=cfg.current_rows,
        psi_bins=cfg.psi_bins,
        psi_threshold=cfg.psi_threshold,
        ks_threshold=cfg.ks_threshold,
        missing_delta_threshold=cfg.missing_delta_threshold,
    )
    summary = {
        "ok": True,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "config": _jsonable(asdict(cfg)),
        "prediction_performance": prediction_block,
        "drift": {
            "drift_detected": drift_report["drift_detected"],
            "drifted_features": drift_report["drifted_features"],
            "max_psi_feature": drift_report["max_psi_feature"],
            "max_ks_feature": drift_report["max_ks_feature"],
            "output": str(cfg.drift_report_output_path),
        },
    }
    cfg.summary_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.summary_output_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return summary


def _run_prediction_performance(cfg: MonitoringCycleConfig) -> dict[str, Any]:
    if not cfg.prediction_log_path.exists() or cfg.prediction_log_path.stat().st_size == 0:
        if cfg.skip_missing_prediction_log:
            return {
                "ok": True,
                "skipped": True,
                "reason": f"prediction log not found: {cfg.prediction_log_path}",
            }
        raise FileNotFoundError(f"prediction log not found: {cfg.prediction_log_path}")

    backfill = backfill_prediction_log(
        prediction_log_path=cfg.prediction_log_path,
        samples_path=cfg.samples_path,
        output_path=cfg.backfilled_log_output_path,
        flat_threshold_pct=cfg.flat_threshold_pct,
        horizons=list(cfg.horizons) if cfg.horizons else None,
    )
    performance = evaluate_prediction_log(
        prediction_log_path=cfg.backfilled_log_output_path,
        output_path=cfg.performance_report_output_path,
        high_confidence_threshold=cfg.high_confidence_threshold,
        round_trip_cost_pct=cfg.round_trip_cost_pct,
    )
    return {
        "ok": True,
        "skipped": False,
        "backfill": backfill,
        "performance": {
            "rows": performance["rows"],
            "labeled_rows": performance["labeled_rows"],
            "overall": performance["overall"],
            "output": str(cfg.performance_report_output_path),
        },
    }


def _jsonable(value: Any) -> Any:
    if isinstance(value, Path):
        return str(value)
    if isinstance(value, dict):
        return {key: _jsonable(item) for key, item in value.items()}
    if isinstance(value, list):
        return [_jsonable(item) for item in value]
    return value


if __name__ == "__main__":
    main()
