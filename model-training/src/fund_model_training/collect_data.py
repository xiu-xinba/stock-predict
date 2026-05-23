from __future__ import annotations

import argparse
import json
from pathlib import Path

from fund_model_training.collectors.akshare_adapter import (
    collect_etf_daily,
    collect_etf_intraday,
    collect_etf_spot,
    collect_etf_universe,
    collect_futures_main,
    collect_index_daily,
    collect_index_intraday,
)
from fund_model_training.collectors.common import write_csv


DATASET_TABLES: dict[str, str] = {
    "etf_universe": "dim_fund",
    "etf_spot": "fund_intraday",
    "etf_daily": "fund_daily",
    "etf_intraday": "fund_intraday",
    "index_daily": "index_daily",
    "index_intraday": "index_intraday",
    "futures_main": "futures_bar",
}


def main() -> None:
    parser = argparse.ArgumentParser(description="Collect phase-1 index-fund datasets into contract CSVs.")
    parser.add_argument("--source", choices=["akshare"], default="akshare")
    parser.add_argument("--dataset", choices=sorted(DATASET_TABLES), required=True)
    parser.add_argument("--symbol", help="Fund, index, or futures symbol when the dataset requires one.")
    parser.add_argument("--underlying", help="Underlying name for futures rows, e.g. IF, IH, IC, IM, Brent.")
    parser.add_argument("--start-date", help="Start date accepted by the source API, usually YYYYMMDD.")
    parser.add_argument("--end-date", help="End date accepted by the source API, usually YYYYMMDD.")
    parser.add_argument("--period", default="1", help="Intraday period, e.g. 1, 5, 15, 30, 60.")
    parser.add_argument("--adjust", default="", help="Adjustment mode accepted by the source API.")
    parser.add_argument("--tracking-map", help="CSV with fund_code,tracking_index[,market] for ETF universe mapping.")
    parser.add_argument("--skip-validation", action="store_true", help="Write raw normalized output even if validation fails.")
    parser.add_argument("--output", type=Path, help="Output CSV path. Defaults to data/raw/{dataset}.csv.")
    args = parser.parse_args()

    if args.source != "akshare":
        raise SystemExit(f"Unsupported source: {args.source}")

    df = _collect_akshare(args)
    output = args.output or Path("data") / "raw" / f"{args.dataset}.csv"
    out = write_csv(df, output)
    print(json.dumps({
        "ok": True,
        "source": args.source,
        "dataset": args.dataset,
        "table": DATASET_TABLES[args.dataset],
        "rows": int(len(df)),
        "output": str(out),
    }, ensure_ascii=False, indent=2))


def _collect_akshare(args):
    if args.dataset == "etf_universe":
        return collect_etf_universe(args.tracking_map, skip_validation=args.skip_validation)
    if args.dataset == "etf_spot":
        return collect_etf_spot(skip_validation=args.skip_validation)
    if args.dataset == "etf_daily":
        _require(args.symbol, "--symbol is required for etf_daily.")
        _require(args.start_date, "--start-date is required for etf_daily.")
        _require(args.end_date, "--end-date is required for etf_daily.")
        return collect_etf_daily(
            symbol=args.symbol,
            start_date=args.start_date,
            end_date=args.end_date,
            adjust=args.adjust,
            skip_validation=args.skip_validation,
        )
    if args.dataset == "etf_intraday":
        _require(args.symbol, "--symbol is required for etf_intraday.")
        return collect_etf_intraday(
            symbol=args.symbol,
            period=args.period,
            start_date=args.start_date,
            end_date=args.end_date,
            adjust=args.adjust,
            skip_validation=args.skip_validation,
        )
    if args.dataset == "index_daily":
        _require(args.symbol, "--symbol is required for index_daily.")
        return collect_index_daily(
            symbol=args.symbol,
            start_date=args.start_date,
            end_date=args.end_date,
            skip_validation=args.skip_validation,
        )
    if args.dataset == "index_intraday":
        _require(args.symbol, "--symbol is required for index_intraday.")
        return collect_index_intraday(
            symbol=args.symbol,
            period=args.period,
            start_date=args.start_date,
            end_date=args.end_date,
            skip_validation=args.skip_validation,
        )
    if args.dataset == "futures_main":
        _require(args.symbol, "--symbol is required for futures_main.")
        _require(args.start_date, "--start-date is required for futures_main.")
        _require(args.end_date, "--end-date is required for futures_main.")
        return collect_futures_main(
            symbol=args.symbol,
            start_date=args.start_date,
            end_date=args.end_date,
            underlying=args.underlying,
            skip_validation=args.skip_validation,
        )
    raise SystemExit(f"Unsupported dataset: {args.dataset}")


def _require(value, message: str) -> None:
    if value is None or value == "":
        raise SystemExit(message)


if __name__ == "__main__":
    main()
