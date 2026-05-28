from __future__ import annotations

import argparse
import json
from pathlib import Path

from fund_model_training.build_index_fund_dataset import _load_universe, _safe_name
from fund_model_training.build_intraday_samples import build_intraday_samples
from fund_model_training.collectors.akshare_adapter import collect_etf_intraday, collect_index_intraday
from fund_model_training.collectors.common import require_pandas, write_csv


def main() -> None:
    parser = argparse.ArgumentParser(description="Build a multi-fund 3/5-minute index-fund dataset.")
    parser.add_argument("--universe", type=Path, required=True, help="CSV with fund_code,tracking_index,market.")
    parser.add_argument("--raw-dir", type=Path, default=Path("data/raw/intraday_batch"))
    parser.add_argument("--history-dir", type=Path, help="Optional accumulated intraday history directory.")
    parser.add_argument("--output", type=Path, default=Path("data/processed/intraday_index_fund_samples.csv"))
    parser.add_argument("--panic-factor", type=Path)
    parser.add_argument("--period", default="1", help="Intraday source period. MVP supports 1-minute bars.")
    parser.add_argument("--start-date", help="Optional source start datetime.")
    parser.add_argument("--end-date", help="Optional source end datetime.")
    parser.add_argument("--horizon-minutes", type=int, default=5, choices=[3, 5])
    parser.add_argument("--max-funds", type=int, help="Limit the number of funds for smoke runs.")
    parser.add_argument("--continue-on-error", action="store_true")
    parser.add_argument("--skip-existing", action="store_true")
    parser.add_argument("--flat-threshold-pct", type=float, default=0.02)
    args = parser.parse_args()

    summary = build_intraday_index_fund_dataset(
        universe_path=args.universe,
        raw_dir=args.raw_dir,
        history_dir=args.history_dir,
        output_path=args.output,
        panic_factor_path=args.panic_factor,
        period=args.period,
        start_date=args.start_date,
        end_date=args.end_date,
        horizon_minutes=args.horizon_minutes,
        max_funds=args.max_funds,
        continue_on_error=args.continue_on_error,
        skip_existing=args.skip_existing,
        flat_threshold_pct=args.flat_threshold_pct,
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def build_intraday_index_fund_dataset(
    universe_path: str | Path,
    raw_dir: str | Path,
    output_path: str | Path,
    history_dir: str | Path | None = None,
    panic_factor_path: str | Path | None = None,
    period: str = "1",
    start_date: str | None = None,
    end_date: str | None = None,
    horizon_minutes: int = 5,
    max_funds: int | None = None,
    continue_on_error: bool = False,
    skip_existing: bool = False,
    flat_threshold_pct: float = 0.02,
) -> dict:
    pd = require_pandas()
    raw_dir = Path(raw_dir)
    raw_dir.mkdir(parents=True, exist_ok=True)
    history_dir = Path(history_dir) if history_dir else None
    if history_dir:
        history_dir.mkdir(parents=True, exist_ok=True)
    universe = _load_universe(universe_path)
    if max_funds:
        universe = universe.head(max_funds).copy()

    dim_path = raw_dir / "dim_fund_intraday_batch.csv"
    dim = universe[["fund_code", "fund_name", "tracking_index", "market"]].copy()
    dim["fund_type"] = "ETF"
    dim["is_etf"] = True
    dim["is_lof"] = False
    dim["fee_rate"] = None
    dim["inception_date"] = None
    dim = dim[["fund_code", "fund_name", "fund_type", "tracking_index", "market", "is_etf", "is_lof", "fee_rate", "inception_date"]]
    write_csv(dim, dim_path)

    errors: list[dict[str, str]] = []
    fund_frames = []
    for _, row in universe.iterrows():
        fund_code = str(row["fund_code"])
        path = raw_dir / f"fund_intraday_{fund_code}.csv"
        try:
            if skip_existing and path.exists():
                fund_df = pd.read_csv(path, dtype={"fund_code": str})
            else:
                fund_df = collect_etf_intraday(
                    symbol=fund_code,
                    period=period,
                    start_date=start_date,
                    end_date=end_date,
                    skip_validation=True,
                )
                write_csv(fund_df, path)
            fund_frames.append(fund_df)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": fund_code, "type": "fund_intraday", "error": str(exc)})

    if not fund_frames:
        raise ValueError("No fund_intraday data collected.")
    fund_intraday_path = raw_dir / "fund_intraday_batch.csv"
    fund_batch = pd.concat(fund_frames, ignore_index=True)
    write_csv(fund_batch, fund_intraday_path)
    if history_dir:
        fund_intraday_path = _append_history(
            new_rows=fund_batch,
            history_path=history_dir / "fund_intraday_history.csv",
            key_columns=["fund_code", "timestamp"],
            dtype={"fund_code": str},
        )

    index_frames = []
    for index_code in sorted(set(universe["tracking_index"].dropna())):
        path = raw_dir / f"index_intraday_{_safe_name(index_code)}.csv"
        try:
            if skip_existing and path.exists():
                index_df = pd.read_csv(path, dtype={"index_code": str})
            else:
                index_df = collect_index_intraday(
                    symbol=index_code,
                    period=period,
                    start_date=start_date,
                    end_date=end_date,
                    skip_validation=True,
                )
                write_csv(index_df, path)
            index_frames.append(index_df)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": index_code, "type": "index_intraday", "error": str(exc)})

    if not index_frames:
        raise ValueError("No index_intraday data collected.")
    index_intraday_path = raw_dir / "index_intraday_batch.csv"
    index_batch = pd.concat(index_frames, ignore_index=True)
    write_csv(index_batch, index_intraday_path)
    if history_dir:
        index_intraday_path = _append_history(
            new_rows=index_batch,
            history_path=history_dir / "index_intraday_history.csv",
            key_columns=["index_code", "timestamp"],
            dtype={"index_code": str},
        )

    sample_frames = []
    for _, row in universe.iterrows():
        fund_code = str(row["fund_code"])
        try:
            samples = build_intraday_samples(
                fund_intraday_path=fund_intraday_path,
                index_intraday_path=index_intraday_path,
                dim_fund_path=dim_path,
                panic_factor_path=panic_factor_path,
                fund_code=fund_code,
                tracking_index=str(row["tracking_index"]),
                market=str(row["market"]),
                horizon_minutes=horizon_minutes,
                flat_threshold_pct=flat_threshold_pct,
            )
            sample_frames.append(samples)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": fund_code, "type": "intraday_samples", "error": str(exc)})

    if not sample_frames:
        raise ValueError("No intraday samples built.")
    output = write_csv(pd.concat(sample_frames, ignore_index=True), output_path)
    return {
        "ok": len(errors) == 0,
        "universe_rows": int(len(universe)),
        "sample_rows": int(sum(len(frame) for frame in sample_frames)),
        "funds_collected": int(len(fund_frames)),
        "indexes_collected": int(len(index_frames)),
        "horizon_minutes": int(horizon_minutes),
        "output": str(output),
        "raw_dir": str(raw_dir),
        "history_dir": str(history_dir) if history_dir else None,
        "errors": errors,
    }


def _append_history(new_rows, history_path: Path, key_columns: list[str], dtype: dict[str, type] | None = None) -> Path:
    pd = require_pandas()
    history_path.parent.mkdir(parents=True, exist_ok=True)
    if history_path.exists():
        old = pd.read_csv(history_path, dtype=dtype)
        combined = pd.concat([old, new_rows], ignore_index=True)
    else:
        combined = new_rows.copy()
    missing = [column for column in key_columns if column not in combined.columns]
    if missing:
        raise ValueError(f"Cannot append intraday history; missing key columns: {', '.join(missing)}")
    combined = combined.drop_duplicates(subset=key_columns, keep="last")
    sort_columns = [column for column in key_columns if column in combined.columns]
    combined = combined.sort_values(sort_columns).reset_index(drop=True)
    write_csv(combined, history_path)
    return history_path


if __name__ == "__main__":
    main()
