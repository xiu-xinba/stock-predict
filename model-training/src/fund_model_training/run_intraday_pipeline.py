from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

from fund_model_training.build_intraday_index_fund_dataset import build_intraday_index_fund_dataset
from fund_model_training.promote_model import PromotionConfig, promote_model
from fund_model_training.train_tournament import TournamentConfig, train_tournament


@dataclass(frozen=True)
class IntradayPipelineConfig:
    universe_path: Path
    raw_dir: Path
    history_dir: Path | None
    samples_output_path: Path
    report_output_path: Path
    metadata_output_path: Path
    champion_output_path: Path
    summary_output_path: Path
    registry_dir: Path
    panic_factor_path: Path | None = None
    period: str = "1"
    start_date: str | None = None
    end_date: str | None = None
    horizon_minutes: int = 5
    max_funds: int | None = None
    continue_on_error: bool = True
    skip_existing: bool = False
    flat_threshold_pct: float = 0.02
    high_confidence_threshold: float = 0.60
    candidates: tuple[str, ...] = TournamentConfig.__dataclass_fields__["candidates"].default
    ablation_tests: tuple[str, ...] = ()
    purged_split: bool = True
    embargo_minutes: int | None = None
    rolling_backtest_folds: int = 3
    rolling_backtest_min_train_rows: int = 500
    rolling_backtest_min_test_rows: int = 100
    min_balanced_accuracy: float = 0.34
    max_regression_mae: float | None = 0.12
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
    parser = argparse.ArgumentParser(description="Run intraday data -> tournament -> promotion pipeline.")
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--max-funds", type=int, help="Override config max_funds.")
    parser.add_argument("--skip-existing", action="store_true")
    parser.add_argument("--allow-lower-score", action="store_true", help="Bootstrap current even if score is not higher.")
    args = parser.parse_args()

    cfg = load_intraday_pipeline_config(args.config)
    if args.max_funds is not None or args.skip_existing or args.allow_lower_score:
        cfg = IntradayPipelineConfig(
            **{
                **asdict(cfg),
                "max_funds": args.max_funds if args.max_funds is not None else cfg.max_funds,
                "skip_existing": args.skip_existing or cfg.skip_existing,
                "allow_lower_score": args.allow_lower_score or cfg.allow_lower_score,
            }
        )
    summary = run_intraday_pipeline(cfg)
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def load_intraday_pipeline_config(path: str | Path) -> IntradayPipelineConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install PyYAML`.") from exc

    config_path = Path(path)
    raw = yaml.safe_load(config_path.read_text(encoding="utf-8")) or {}
    base_dir = config_path.parent.parent

    def resolve(value: str | None) -> Path | None:
        if value is None or value == "":
            return None
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    promotion = raw.get("promotion") or {}
    return IntradayPipelineConfig(
        universe_path=resolve(str(raw["universe_path"])),
        raw_dir=resolve(str(raw.get("raw_dir", "data/raw/public_mvp_intraday"))),
        history_dir=resolve(raw.get("history_dir")),
        samples_output_path=resolve(str(raw.get("samples_output_path", "data/processed/public_mvp_intraday_index_fund_samples.csv"))),
        panic_factor_path=resolve(raw.get("panic_factor_path")),
        report_output_path=resolve(str(raw.get("report_output_path", "reports/public_mvp_index_fund_intraday_tournament_report.json"))),
        metadata_output_path=resolve(str(raw.get("metadata_output_path", "artifacts/public_mvp_index_fund_intraday_tournament_metadata.json"))),
        champion_output_path=resolve(str(raw.get("champion_output_path", "artifacts/public_mvp_index_fund_intraday_tournament_champion.joblib"))),
        summary_output_path=resolve(str(raw.get("summary_output_path", "reports/public_mvp_intraday_pipeline_summary.json"))),
        registry_dir=resolve(str(promotion.get("registry_dir", raw.get("registry_dir", "model_registry")))),
        period=str(raw.get("period", "1")),
        start_date=raw.get("start_date"),
        end_date=raw.get("end_date"),
        horizon_minutes=int(raw.get("horizon_minutes", 5)),
        max_funds=int(raw["max_funds"]) if raw.get("max_funds") is not None else None,
        continue_on_error=bool(raw.get("continue_on_error", True)),
        skip_existing=bool(raw.get("skip_existing", False)),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.02)),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
        candidates=tuple(raw.get("candidates", TournamentConfig.__dataclass_fields__["candidates"].default)),
        ablation_tests=tuple(raw.get("ablation_tests", ())),
        purged_split=bool(raw.get("purged_split", True)),
        embargo_minutes=_optional_int(raw.get("embargo_minutes")),
        rolling_backtest_folds=int(raw.get("rolling_backtest_folds", 3)),
        rolling_backtest_min_train_rows=int(raw.get("rolling_backtest_min_train_rows", 500)),
        rolling_backtest_min_test_rows=int(raw.get("rolling_backtest_min_test_rows", 100)),
        min_balanced_accuracy=float(promotion.get("min_balanced_accuracy", 0.34)),
        max_regression_mae=_optional_float(promotion.get("max_regression_mae", 0.12)),
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


def run_intraday_pipeline(cfg: IntradayPipelineConfig) -> dict:
    dataset = build_intraday_index_fund_dataset(
        universe_path=cfg.universe_path,
        raw_dir=cfg.raw_dir,
        history_dir=cfg.history_dir,
        output_path=cfg.samples_output_path,
        panic_factor_path=cfg.panic_factor_path,
        period=cfg.period,
        start_date=cfg.start_date,
        end_date=cfg.end_date,
        horizon_minutes=cfg.horizon_minutes,
        max_funds=cfg.max_funds,
        continue_on_error=cfg.continue_on_error,
        skip_existing=cfg.skip_existing,
        flat_threshold_pct=cfg.flat_threshold_pct,
    )
    target_col = f"future_return_pct_{cfg.horizon_minutes}m"
    train_metadata = train_tournament(TournamentConfig(
        task=f"public_mvp_index_fund_intraday_{cfg.horizon_minutes}m_tournament",
        data_path=cfg.samples_output_path,
        report_output_path=cfg.report_output_path,
        metadata_output_path=cfg.metadata_output_path,
        champion_output_path=cfg.champion_output_path,
        feature_set="index_fund_intraday_v1",
        future_return_column=target_col,
        regression_target=target_col,
        flat_threshold_pct=cfg.flat_threshold_pct,
        high_confidence_threshold=cfg.high_confidence_threshold,
        candidates=cfg.candidates,
        ablation_tests=cfg.ablation_tests,
        purged_split=cfg.purged_split,
        embargo_minutes=cfg.embargo_minutes,
        rolling_backtest_folds=cfg.rolling_backtest_folds,
        rolling_backtest_min_train_rows=cfg.rolling_backtest_min_train_rows,
        rolling_backtest_min_test_rows=cfg.rolling_backtest_min_test_rows,
    ))
    promotion = promote_model(PromotionConfig(
        task=f"intraday_{cfg.horizon_minutes}m",
        challenger_model_path=cfg.champion_output_path,
        challenger_metadata_path=cfg.metadata_output_path,
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
        "ok": bool(dataset["sample_rows"] > 0),
        "dataset": dataset,
        "train": train_metadata,
        "promotion": promotion,
        "outputs": {
            "samples": str(cfg.samples_output_path),
            "report": str(cfg.report_output_path),
            "metadata": str(cfg.metadata_output_path),
            "champion": str(cfg.champion_output_path),
        },
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


if __name__ == "__main__":
    main()
