from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from fund_model_training.promote_model import PromotionConfig, promote_model
from fund_model_training.train_tournament import load_tournament_config, train_tournament


@dataclass(frozen=True)
class RetrainingCycleConfig:
    task: str
    tournament_config_path: Path
    registry_dir: Path = Path("model_registry")
    summary_output_path: Path = Path("reports/retraining_cycle_summary.json")
    min_balanced_accuracy: float = 0.34
    max_regression_mae: float | None = None
    min_high_confidence_accuracy: float | None = None
    min_high_confidence_coverage: float = 0.0
    min_train_rows: int | None = None
    min_test_rows: int | None = None
    min_calendar_days: int | None = None
    min_score_delta: float = 0.0
    max_high_confidence_accuracy_drop: float | None = None
    max_high_confidence_coverage_drop: float | None = None
    min_market_regime_slice_rows: int | None = None
    min_market_regime_accuracy: float | None = None
    min_rolling_backtest_folds: int | None = None
    min_rolling_balanced_accuracy: float | None = None
    mae_weight: float = 0.05
    allow_lower_score: bool = False


def main() -> None:
    parser = argparse.ArgumentParser(description="Run train -> evaluate -> promote for one model task.")
    parser.add_argument("--config", type=Path, required=True, help="Retraining cycle YAML config.")
    args = parser.parse_args()

    cfg = load_retraining_cycle_config(args.config)
    summary = run_retraining_cycle(cfg)
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def load_retraining_cycle_config(path: str | Path) -> RetrainingCycleConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install PyYAML`.") from exc

    config_path = Path(path)
    raw = yaml.safe_load(config_path.read_text(encoding="utf-8")) or {}
    base_dir = config_path.parent.parent

    def resolve(value: str | Path) -> Path:
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    promotion = raw.get("promotion") or {}
    return RetrainingCycleConfig(
        task=str(raw["task"]),
        tournament_config_path=resolve(raw["tournament_config_path"]),
        registry_dir=resolve(promotion.get("registry_dir", raw.get("registry_dir", "model_registry"))),
        summary_output_path=resolve(raw.get("summary_output_path", "reports/retraining_cycle_summary.json")),
        min_balanced_accuracy=float(promotion.get("min_balanced_accuracy", 0.34)),
        max_regression_mae=_optional_float(promotion.get("max_regression_mae")),
        min_high_confidence_accuracy=_optional_float(promotion.get("min_high_confidence_accuracy")),
        min_high_confidence_coverage=float(promotion.get("min_high_confidence_coverage", 0.0)),
        min_train_rows=_optional_int(promotion.get("min_train_rows")),
        min_test_rows=_optional_int(promotion.get("min_test_rows")),
        min_calendar_days=_optional_int(promotion.get("min_calendar_days")),
        min_score_delta=float(promotion.get("min_score_delta", 0.0)),
        max_high_confidence_accuracy_drop=_optional_float(promotion.get("max_high_confidence_accuracy_drop")),
        max_high_confidence_coverage_drop=_optional_float(promotion.get("max_high_confidence_coverage_drop")),
        min_market_regime_slice_rows=_optional_int(promotion.get("min_market_regime_slice_rows")),
        min_market_regime_accuracy=_optional_float(promotion.get("min_market_regime_accuracy")),
        min_rolling_backtest_folds=_optional_int(promotion.get("min_rolling_backtest_folds")),
        min_rolling_balanced_accuracy=_optional_float(promotion.get("min_rolling_balanced_accuracy")),
        mae_weight=float(promotion.get("mae_weight", 0.05)),
        allow_lower_score=bool(promotion.get("allow_lower_score", False)),
    )


def run_retraining_cycle(cfg: RetrainingCycleConfig) -> dict[str, Any]:
    tournament_cfg = load_tournament_config(cfg.tournament_config_path)
    train_metadata = train_tournament(tournament_cfg)
    promotion_report = promote_model(PromotionConfig(
        task=cfg.task,
        challenger_model_path=tournament_cfg.champion_output_path,
        challenger_metadata_path=tournament_cfg.metadata_output_path,
        registry_dir=cfg.registry_dir,
        min_balanced_accuracy=cfg.min_balanced_accuracy,
        max_regression_mae=cfg.max_regression_mae,
        min_high_confidence_accuracy=cfg.min_high_confidence_accuracy,
        min_high_confidence_coverage=cfg.min_high_confidence_coverage,
        min_train_rows=cfg.min_train_rows,
        min_test_rows=cfg.min_test_rows,
        min_calendar_days=cfg.min_calendar_days,
        min_score_delta=cfg.min_score_delta,
        max_high_confidence_accuracy_drop=cfg.max_high_confidence_accuracy_drop,
        max_high_confidence_coverage_drop=cfg.max_high_confidence_coverage_drop,
        min_market_regime_slice_rows=cfg.min_market_regime_slice_rows,
        min_market_regime_accuracy=cfg.min_market_regime_accuracy,
        min_rolling_backtest_folds=cfg.min_rolling_backtest_folds,
        min_rolling_balanced_accuracy=cfg.min_rolling_balanced_accuracy,
        mae_weight=cfg.mae_weight,
        allow_lower_score=cfg.allow_lower_score,
    ))
    summary = {
        "task": cfg.task,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "config": _jsonable(asdict(cfg)),
        "train": train_metadata,
        "promotion": promotion_report,
    }
    cfg.summary_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.summary_output_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return summary


def _optional_float(value: Any) -> float | None:
    if value is None or value == "":
        return None
    return float(value)


def _optional_int(value: Any) -> int | None:
    if value is None or value == "":
        return None
    return int(value)


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
