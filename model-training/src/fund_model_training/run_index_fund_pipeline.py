from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path

from fund_model_training.build_index_fund_dataset import build_index_fund_dataset
from fund_model_training.build_panic_proxy import build_panic_proxy_components
from fund_model_training.build_panic_factor import DEFAULT_WEIGHTS, build_panic_factor_from_frame
from fund_model_training.collectors.common import write_csv
from fund_model_training.train_baseline import BaselineConfig, train_baseline


@dataclass(frozen=True)
class PipelineConfig:
    universe_path: Path
    start_date: str
    end_date: str
    raw_dir: Path
    samples_output_path: Path
    pre_panic_samples_output_path: Path
    panic_components_output_path: Path
    panic_factor_output_path: Path
    report_output_path: Path
    metadata_output_path: Path
    model_output_path: Path
    summary_output_path: Path
    max_funds: int | None = None
    continue_on_error: bool = True
    skip_existing: bool = False
    market: str = "CN"
    flat_threshold_pct: float = 0.05
    high_confidence_threshold: float = 0.60


def main() -> None:
    parser = argparse.ArgumentParser(description="Run the public-data index-fund MVP pipeline end to end.")
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--max-funds", type=int, help="Override config max_funds for smoke runs.")
    parser.add_argument("--skip-existing", action="store_true", help="Reuse existing raw symbol files.")
    args = parser.parse_args()

    cfg = load_pipeline_config(args.config)
    if args.max_funds is not None or args.skip_existing:
        cfg = PipelineConfig(
            **{
                **asdict(cfg),
                "max_funds": args.max_funds if args.max_funds is not None else cfg.max_funds,
                "skip_existing": args.skip_existing or cfg.skip_existing,
            }
        )
    summary = run_pipeline(cfg)
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def load_pipeline_config(path: str | Path) -> PipelineConfig:
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

    return PipelineConfig(
        universe_path=resolve(str(raw["universe_path"])),
        start_date=str(raw["start_date"]),
        end_date=str(raw["end_date"]),
        raw_dir=resolve(str(raw.get("raw_dir", "data/raw/public_mvp"))),
        samples_output_path=resolve(str(raw.get("samples_output_path", "data/processed/public_mvp_daily_weekly_index_fund_samples.csv"))),
        pre_panic_samples_output_path=resolve(str(raw.get("pre_panic_samples_output_path", "data/processed/public_mvp_pre_panic_samples.csv"))),
        panic_components_output_path=resolve(str(raw.get("panic_components_output_path", "data/processed/public_mvp_panic_components.csv"))),
        panic_factor_output_path=resolve(str(raw.get("panic_factor_output_path", "data/processed/public_mvp_panic_factor.csv"))),
        report_output_path=resolve(str(raw.get("report_output_path", "reports/public_mvp_index_fund_baseline_report.json"))),
        metadata_output_path=resolve(str(raw.get("metadata_output_path", "artifacts/public_mvp_index_fund_baseline_metadata.json"))),
        model_output_path=resolve(str(raw.get("model_output_path", "artifacts/public_mvp_index_fund_baseline.joblib"))),
        summary_output_path=resolve(str(raw.get("summary_output_path", "reports/public_mvp_pipeline_summary.json"))),
        max_funds=int(raw["max_funds"]) if raw.get("max_funds") is not None else None,
        continue_on_error=bool(raw.get("continue_on_error", True)),
        skip_existing=bool(raw.get("skip_existing", False)),
        market=str(raw.get("market", "CN")),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.05)),
        high_confidence_threshold=float(raw.get("high_confidence_threshold", 0.60)),
    )


def run_pipeline(cfg: PipelineConfig) -> dict:
    first_pass = build_index_fund_dataset(
        universe_path=cfg.universe_path,
        start_date=cfg.start_date,
        end_date=cfg.end_date,
        raw_dir=cfg.raw_dir,
        output_path=cfg.pre_panic_samples_output_path,
        panic_factor_path=None,
        max_funds=cfg.max_funds,
        continue_on_error=cfg.continue_on_error,
        skip_existing=cfg.skip_existing,
        flat_threshold_pct=cfg.flat_threshold_pct,
    )

    index_daily_path = cfg.raw_dir / "index_daily_batch.csv"
    futures_path = cfg.raw_dir / "futures_batch.csv"
    components = build_panic_proxy_components(index_daily_path, futures_path=futures_path, market=cfg.market)
    write_csv(components, cfg.panic_components_output_path)
    panic = build_panic_factor_from_frame(
        raw=components,
        market=cfg.market,
        timestamp_col="timestamp",
        available_time_col="available_time",
        weights=DEFAULT_WEIGHTS,
    )
    write_csv(panic, cfg.panic_factor_output_path)

    final_pass = build_index_fund_dataset(
        universe_path=cfg.universe_path,
        start_date=cfg.start_date,
        end_date=cfg.end_date,
        raw_dir=cfg.raw_dir,
        output_path=cfg.samples_output_path,
        panic_factor_path=cfg.panic_factor_output_path,
        max_funds=cfg.max_funds,
        continue_on_error=cfg.continue_on_error,
        skip_existing=True,
        flat_threshold_pct=cfg.flat_threshold_pct,
    )

    metadata = train_baseline(BaselineConfig(
        task="public_mvp_index_fund_daily_baseline",
        data_path=cfg.samples_output_path,
        report_output_path=cfg.report_output_path,
        metadata_output_path=cfg.metadata_output_path,
        model_output_path=cfg.model_output_path,
        feature_set="index_fund_daily_v1",
        flat_threshold_pct=cfg.flat_threshold_pct,
        high_confidence_threshold=cfg.high_confidence_threshold,
    ))

    summary = {
        "ok": final_pass["ok"],
        "first_pass": first_pass,
        "final_pass": final_pass,
        "panic_rows": int(len(panic)),
        "training": metadata,
        "outputs": {
            "samples": str(cfg.samples_output_path),
            "panic_factor": str(cfg.panic_factor_output_path),
            "report": str(cfg.report_output_path),
            "metadata": str(cfg.metadata_output_path),
            "model": str(cfg.model_output_path),
        },
    }
    cfg.summary_output_path.parent.mkdir(parents=True, exist_ok=True)
    cfg.summary_output_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return summary


if __name__ == "__main__":
    main()
