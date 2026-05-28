from __future__ import annotations

import argparse
import json
from pathlib import Path

from fund_model_training.build_samples import build_daily_weekly_samples
from fund_model_training.collectors.akshare_adapter import collect_etf_daily, collect_futures_main, collect_index_daily
from fund_model_training.collectors.common import require_pandas, standardize_code, write_csv


def main() -> None:
    parser = argparse.ArgumentParser(description="Build a multi-fund index-fund dataset from a universe CSV.")
    parser.add_argument("--universe", type=Path, required=True, help="CSV with fund_code,tracking_index,market,futures_symbol,futures_underlying.")
    parser.add_argument("--start-date", required=True, help="Start date for source APIs, usually YYYYMMDD.")
    parser.add_argument("--end-date", required=True, help="End date for source APIs, usually YYYYMMDD.")
    parser.add_argument("--raw-dir", type=Path, default=Path("data/raw/batch"))
    parser.add_argument("--output", type=Path, default=Path("data/processed/daily_weekly_index_fund_samples.csv"))
    parser.add_argument("--panic-factor", type=Path)
    parser.add_argument("--max-funds", type=int, help="Limit the number of funds for smoke runs.")
    parser.add_argument("--continue-on-error", action="store_true", help="Skip failed symbols and continue.")
    parser.add_argument("--skip-existing", action="store_true", help="Reuse existing raw per-symbol files.")
    parser.add_argument("--flat-threshold-pct", type=float, default=0.05)
    args = parser.parse_args()

    summary = build_index_fund_dataset(
        universe_path=args.universe,
        start_date=args.start_date,
        end_date=args.end_date,
        raw_dir=args.raw_dir,
        output_path=args.output,
        panic_factor_path=args.panic_factor,
        max_funds=args.max_funds,
        continue_on_error=args.continue_on_error,
        skip_existing=args.skip_existing,
        flat_threshold_pct=args.flat_threshold_pct,
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2))


def build_index_fund_dataset(
    universe_path: str | Path,
    start_date: str,
    end_date: str,
    raw_dir: str | Path,
    output_path: str | Path,
    panic_factor_path: str | Path | None = None,
    max_funds: int | None = None,
    continue_on_error: bool = False,
    skip_existing: bool = False,
    flat_threshold_pct: float = 0.05,
) -> dict:
    pd = require_pandas()
    raw_dir = Path(raw_dir)
    raw_dir.mkdir(parents=True, exist_ok=True)
    universe = _load_universe(universe_path)
    if max_funds:
        universe = universe.head(max_funds).copy()

    dim_path = raw_dir / "dim_fund_batch.csv"
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
        fund_code = row["fund_code"]
        path = raw_dir / f"fund_daily_{fund_code}.csv"
        try:
            if skip_existing and path.exists():
                fund_df = pd.read_csv(path, dtype={"fund_code": str})
            else:
                fund_df = collect_etf_daily(fund_code, start_date=start_date, end_date=end_date)
                write_csv(fund_df, path)
            fund_frames.append(fund_df)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": fund_code, "type": "fund_daily", "error": str(exc)})

    if not fund_frames:
        raise ValueError("No fund_daily data collected.")
    fund_daily_path = raw_dir / "fund_daily_batch.csv"
    write_csv(pd.concat(fund_frames, ignore_index=True), fund_daily_path)

    index_frames = []
    for index_code in sorted(set(universe["tracking_index"].dropna())):
        path = raw_dir / f"index_daily_{_safe_name(index_code)}.csv"
        try:
            if skip_existing and path.exists():
                index_df = pd.read_csv(path, dtype={"index_code": str})
            else:
                index_df = collect_index_daily(index_code, start_date=start_date, end_date=end_date)
                write_csv(index_df, path)
            index_frames.append(index_df)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": index_code, "type": "index_daily", "error": str(exc)})

    if not index_frames:
        raise ValueError("No index_daily data collected.")
    index_daily_path = raw_dir / "index_daily_batch.csv"
    write_csv(pd.concat(index_frames, ignore_index=True), index_daily_path)

    futures_frames = []
    futures_rows = universe.dropna(subset=["futures_symbol"])
    for _, row in futures_rows.drop_duplicates("futures_symbol").iterrows():
        symbol = str(row["futures_symbol"]).strip()
        if not symbol:
            continue
        path = raw_dir / f"futures_{_safe_name(symbol)}.csv"
        try:
            if skip_existing and path.exists():
                futures_df = pd.read_csv(path)
            else:
                futures_df = collect_futures_main(
                    symbol=symbol,
                    start_date=start_date,
                    end_date=end_date,
                    underlying=str(row.get("futures_underlying", "") or symbol),
                )
                write_csv(futures_df, path)
            futures_frames.append(futures_df)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": symbol, "type": "futures_bar", "error": str(exc)})

    futures_path = None
    if futures_frames:
        futures_path = raw_dir / "futures_batch.csv"
        write_csv(pd.concat(futures_frames, ignore_index=True), futures_path)

    sample_frames = []
    for _, row in universe.iterrows():
        fund_code = row["fund_code"]
        try:
            samples = build_daily_weekly_samples(
                fund_daily_path=fund_daily_path,
                dim_fund_path=dim_path,
                index_daily_path=index_daily_path,
                futures_path=futures_path,
                panic_factor_path=panic_factor_path,
                fund_code=fund_code,
                tracking_index=row["tracking_index"],
                market=row["market"],
                futures_underlying=str(row.get("futures_underlying", "") or "") or None,
                flat_threshold_pct=flat_threshold_pct,
            )
            sample_frames.append(samples)
        except Exception as exc:
            if not continue_on_error:
                raise
            errors.append({"symbol": fund_code, "type": "samples", "error": str(exc)})

    if not sample_frames:
        raise ValueError("No samples built.")
    output = write_csv(pd.concat(sample_frames, ignore_index=True), output_path)
    return {
        "ok": len(errors) == 0,
        "universe_rows": int(len(universe)),
        "sample_rows": int(sum(len(frame) for frame in sample_frames)),
        "funds_collected": int(len(fund_frames)),
        "indexes_collected": int(len(index_frames)),
        "futures_collected": int(len(futures_frames)),
        "output": str(output),
        "raw_dir": str(raw_dir),
        "errors": errors,
    }


def _load_universe(path: str | Path):
    pd = require_pandas()
    df = pd.read_csv(path, dtype=str).fillna("")
    required = {"fund_code", "tracking_index", "market"}
    missing = sorted(required - set(df.columns))
    if missing:
        raise ValueError(f"Universe missing required columns: {', '.join(missing)}")
    if "fund_name" not in df.columns:
        df["fund_name"] = df["fund_code"]
    if "futures_symbol" not in df.columns:
        df["futures_symbol"] = ""
    if "futures_underlying" not in df.columns:
        df["futures_underlying"] = ""
    df["fund_code"] = df["fund_code"].map(standardize_code)
    for column in ("fund_name", "tracking_index", "market", "futures_symbol", "futures_underlying"):
        df[column] = df[column].astype(str).str.strip()
    return df


def _safe_name(value: str) -> str:
    return "".join(ch if ch.isalnum() else "_" for ch in str(value))


if __name__ == "__main__":
    main()
