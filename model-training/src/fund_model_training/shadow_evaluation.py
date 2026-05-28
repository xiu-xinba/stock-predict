from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

from fund_model_training.collectors.common import require_pandas
from fund_model_training.prediction_logging import _performance_block


@dataclass(frozen=True)
class ShadowEvaluationConfig:
    prediction_log_path: Path
    output_path: Path
    challenger_model_version: str
    champion_model_version: str | None = None
    min_shadow_days: int = 5
    min_labeled_rows: int = 20
    min_direction_accuracy: float = 0.0
    min_high_confidence_coverage: float = 0.0
    min_cost_adjusted_return: float | None = None
    high_confidence_threshold: float = 0.60
    round_trip_cost_pct: float = 0.0
    skip_missing_prediction_log: bool = False


def main() -> None:
    parser = argparse.ArgumentParser(description="Evaluate a shadow/challenger model from backfilled prediction logs.")
    parser.add_argument("--config", type=Path, help="YAML config path.")
    parser.add_argument("--prediction-log", type=Path, help="Backfilled prediction_log CSV.")
    parser.add_argument("--output", type=Path, default=Path("reports/shadow_evaluation_report.json"))
    parser.add_argument("--challenger-model-version", help="Exact model_version value for the shadow/challenger rows.")
    parser.add_argument("--champion-model-version", help="Optional exact model_version value for current champion rows.")
    parser.add_argument("--min-shadow-days", type=int, default=5)
    parser.add_argument("--min-labeled-rows", type=int, default=20)
    parser.add_argument("--min-direction-accuracy", type=float, default=0.0)
    parser.add_argument("--min-high-confidence-coverage", type=float, default=0.0)
    parser.add_argument("--min-cost-adjusted-return", type=float)
    parser.add_argument("--high-confidence-threshold", type=float, default=0.60)
    parser.add_argument("--round-trip-cost-pct", type=float, default=0.0)
    args = parser.parse_args()

    cfg = load_shadow_evaluation_config(args.config) if args.config else ShadowEvaluationConfig(
        prediction_log_path=_require_path(args.prediction_log, "--prediction-log is required without --config."),
        output_path=args.output,
        challenger_model_version=_require_text(args.challenger_model_version, "--challenger-model-version is required without --config."),
        champion_model_version=args.champion_model_version,
        min_shadow_days=args.min_shadow_days,
        min_labeled_rows=args.min_labeled_rows,
        min_direction_accuracy=args.min_direction_accuracy,
        min_high_confidence_coverage=args.min_high_confidence_coverage,
        min_cost_adjusted_return=args.min_cost_adjusted_return,
        high_confidence_threshold=args.high_confidence_threshold,
        round_trip_cost_pct=args.round_trip_cost_pct,
    )
    report = evaluate_shadow_model(cfg)
    print(json.dumps(report, ensure_ascii=False, indent=2))


def load_shadow_evaluation_config(path: str | Path) -> ShadowEvaluationConfig:
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

    return ShadowEvaluationConfig(
        prediction_log_path=resolve(raw["prediction_log_path"]),
        output_path=resolve(raw.get("output_path", "reports/shadow_evaluation_report.json")),
        challenger_model_version=str(raw["challenger_model_version"]),
        champion_model_version=str(raw["champion_model_version"]) if raw.get("champion_model_version") else None,
        min_shadow_days=int(raw.get("min_shadow_days", 5)),
        min_labeled_rows=int(raw.get("min_labeled_rows", 20)),
        min_direction_accuracy=float(raw.get("min_direction_accuracy", 0.0)),
        min_high_confidence_coverage=float(raw.get("min_high_confidence_coverage", 0.0)),
        min_cost_adjusted_return=_optional_float(raw.get("min_cost_adjusted_return")),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
        round_trip_cost_pct=float(raw.get("round_trip_cost_pct", 0.0)),
        skip_missing_prediction_log=bool(raw.get("skip_missing_prediction_log", False)),
    )


def evaluate_shadow_model(cfg: ShadowEvaluationConfig) -> dict[str, Any]:
    pd = require_pandas()
    if not cfg.prediction_log_path.exists() or cfg.prediction_log_path.stat().st_size == 0:
        if cfg.skip_missing_prediction_log:
            report = {
                "ok": True,
                "passed": False,
                "skipped": True,
                "reason": f"prediction log not found: {cfg.prediction_log_path}",
                "config": _jsonable(asdict(cfg)),
            }
            _write_report(cfg.output_path, report)
            return report
        raise FileNotFoundError(f"prediction log not found: {cfg.prediction_log_path}")
    df = pd.read_csv(cfg.prediction_log_path, dtype={"fund_code": str, "prediction_id": str})
    labeled = df.dropna(subset=["actual_direction", "actual_return"]).copy()
    labeled["asof_time"] = pd.to_datetime(labeled["asof_time"], errors="coerce")
    labeled["hit_direction"] = labeled["predicted_direction"].astype(str) == labeled["actual_direction"].astype(str)
    labeled["absolute_error_pct"] = (
        pd.to_numeric(labeled["predicted_return"], errors="coerce") -
        pd.to_numeric(labeled["actual_return"], errors="coerce")
    ).abs()

    challenger = labeled.loc[labeled["model_version"].astype(str) == cfg.challenger_model_version].copy()
    champion = (
        labeled.loc[labeled["model_version"].astype(str) == cfg.champion_model_version].copy()
        if cfg.champion_model_version else None
    )
    challenger_block = _model_block(challenger, cfg)
    champion_block = _model_block(champion, cfg) if champion is not None else None
    reasons = _gate_reasons(cfg, challenger_block, champion_block)
    report = {
        "ok": True,
        "passed": not reasons,
        "skipped": False,
        "reasons": reasons,
        "config": _jsonable(asdict(cfg)),
        "challenger": challenger_block,
        "champion": champion_block,
    }
    _write_report(cfg.output_path, report)
    return report


def _write_report(output_path: Path, report: dict[str, Any]) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")


def _model_block(df, cfg: ShadowEvaluationConfig) -> dict[str, Any]:
    if df is None:
        return {"rows": 0, "shadow_days": 0, "performance": None}
    valid_times = df["asof_time"].dropna()
    return {
        "rows": int(len(df)),
        "shadow_days": int(valid_times.dt.date.nunique()) if len(valid_times) else 0,
        "first_asof_time": str(valid_times.min()) if len(valid_times) else None,
        "last_asof_time": str(valid_times.max()) if len(valid_times) else None,
        "performance": _performance_block(df, cfg.high_confidence_threshold, cfg.round_trip_cost_pct),
    }


def _gate_reasons(
    cfg: ShadowEvaluationConfig,
    challenger: dict[str, Any],
    champion: dict[str, Any] | None,
) -> list[str]:
    reasons: list[str] = []
    performance = challenger.get("performance") or {}
    paper = performance.get("paper_trading") or {}
    direction_accuracy = performance.get("direction_accuracy")
    high_conf_coverage = performance.get("high_confidence_coverage")
    cost_adjusted_return = paper.get("mean_cost_adjusted_return")
    if challenger["rows"] < cfg.min_labeled_rows:
        reasons.append(f"labeled_rows below threshold: {challenger['rows']} < {cfg.min_labeled_rows}")
    if challenger["shadow_days"] < cfg.min_shadow_days:
        reasons.append(f"shadow_days below threshold: {challenger['shadow_days']} < {cfg.min_shadow_days}")
    if direction_accuracy is None or direction_accuracy < cfg.min_direction_accuracy:
        reasons.append(f"direction_accuracy below threshold: {direction_accuracy} < {cfg.min_direction_accuracy}")
    if high_conf_coverage is None or high_conf_coverage < cfg.min_high_confidence_coverage:
        reasons.append(
            f"high_confidence_coverage below threshold: {high_conf_coverage} < {cfg.min_high_confidence_coverage}"
        )
    if cfg.min_cost_adjusted_return is not None and (
        cost_adjusted_return is None or cost_adjusted_return < cfg.min_cost_adjusted_return
    ):
        reasons.append(
            f"mean_cost_adjusted_return below threshold: {cost_adjusted_return} < {cfg.min_cost_adjusted_return}"
        )
    if champion and champion.get("performance"):
        champion_perf = champion["performance"]
        champion_paper = champion_perf.get("paper_trading") or {}
        champion_accuracy = champion_perf.get("direction_accuracy")
        champion_return = champion_paper.get("mean_cost_adjusted_return")
        if champion_accuracy is not None and direction_accuracy is not None and direction_accuracy < champion_accuracy:
            reasons.append(f"challenger direction_accuracy below champion: {direction_accuracy} < {champion_accuracy}")
        if champion_return is not None and cost_adjusted_return is not None and cost_adjusted_return < champion_return:
            reasons.append(
                f"challenger mean_cost_adjusted_return below champion: {cost_adjusted_return} < {champion_return}"
            )
    return reasons


def _require_path(value: Path | None, message: str) -> Path:
    if value is None:
        raise SystemExit(message)
    return value


def _require_text(value: str | None, message: str) -> str:
    if value is None or value == "":
        raise SystemExit(message)
    return value


def _optional_float(value: Any) -> float | None:
    if value is None or value == "":
        return None
    return float(value)


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
