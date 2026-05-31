from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np

from fund_model_training.collectors.common import require_pandas, write_csv
from fund_model_training.labels import labels_from_future_return


def main() -> None:
    parser = argparse.ArgumentParser(description="Build all-fund NAV daily training samples.")
    parser.add_argument("--fund-daily", type=Path, required=True, help="fund_daily contract CSV (all fund types).")
    parser.add_argument("--output", type=Path, required=True, help="Output processed sample CSV.")
    parser.add_argument("--fund-code", help="Filter to one fund code.")
    parser.add_argument("--min-samples", type=int, default=120, help="Minimum historical samples per fund (default: 120).")
    parser.add_argument("--max-stale-days", type=int, default=10, help="Max days since last sample (default: 10).")
    parser.add_argument("--flat-threshold-pct", type=float, default=0.05)
    args = parser.parse_args()

    df = build_all_fund_nav_samples(
        fund_daily_path=args.fund_daily,
        fund_code=args.fund_code,
        min_samples=args.min_samples,
        max_stale_days=args.max_stale_days,
        flat_threshold_pct=args.flat_threshold_pct,
    )
    out = write_csv(df, args.output)
    report = {
        "ok": True,
        "rows": int(len(df)),
        "output": str(out),
        "feature_set": "all_fund_nav_daily_v1",
    }
    print(json.dumps(report, ensure_ascii=False, indent=2))


def build_all_fund_nav_samples(
    fund_daily_path: str | Path,
    fund_code: str | None = None,
    min_samples: int = 120,
    max_stale_days: int = 10,
    flat_threshold_pct: float = 0.05,
):
    pd = require_pandas()
    df = pd.read_csv(fund_daily_path, low_memory=False)

    if "fund_code" not in df.columns:
        raise ValueError("fund_daily CSV must contain 'fund_code' column")

    if fund_code:
        df = df.loc[df["fund_code"] == fund_code]

    if "fund_type" not in df.columns:
        df["fund_type"] = "unknown"

    money_mask = df["fund_type"].str.contains("货币", na=False)
    df = df.loc[~money_mask]

    if "nav" in df.columns:
        value_col = "nav"
    elif "adjusted_nav" in df.columns:
        value_col = "adjusted_nav"
    else:
        raise ValueError("fund_daily CSV must contain 'nav' or 'adjusted_nav' column")

    df["asof_time"] = pd.to_datetime(df.get("asof_time", df.get("trade_date", pd.NaT)), errors="coerce")
    df = df.dropna(subset=["asof_time"]).sort_values(["fund_code", "asof_time"])

    df["return_1d"] = df.groupby("fund_code")[value_col].pct_change(1) * 100.0
    df["return_5d"] = df.groupby("fund_code")[value_col].pct_change(5) * 100.0
    df["return_20d"] = df.groupby("fund_code")[value_col].pct_change(20) * 100.0
    df["volatility_20d"] = df.groupby("fund_code")["return_1d"].transform(lambda s: s.rolling(20, min_periods=3).std())
    rolling_max = df.groupby("fund_code")[value_col].transform(lambda s: s.rolling(20, min_periods=3).max())
    df["drawdown_20d"] = ((df[value_col] - rolling_max) / rolling_max.replace(0, np.nan) * 100.0).replace([np.inf, -np.inf], np.nan).fillna(0.0)
    df["mean_reversion"] = -df.groupby("fund_code")["return_1d"].transform(lambda s: s.rolling(5, min_periods=2).mean())

    df["future_return_pct_next_day"] = df.groupby("fund_code")[value_col].pct_change(-1) * 100.0
    df["future_return_pct_1w"] = df.groupby("fund_code")[value_col].pct_change(-5) * 100.0

    df["label_1d"] = labels_from_future_return(df["future_return_pct_next_day"], flat_threshold_pct)
    df["label_1w"] = labels_from_future_return(df["future_return_pct_1w"], flat_threshold_pct)

    sample_counts = df.groupby("fund_code").size()
    eligible_funds = sample_counts[sample_counts >= min_samples].index
    df = df.loc[df["fund_code"].isin(eligible_funds)]

    cutoff = pd.Timestamp.now()
    last_dates = df.groupby("fund_code")["asof_time"].max()
    stale_funds = last_dates[last_dates < cutoff - pd.Timedelta(days=max_stale_days)].index
    df = df.loc[~df["fund_code"].isin(stale_funds)]

    output_cols = [
        "fund_code", "fund_type", "asof_time", "nav",
        "fund_return_1d", "fund_return_5d", "fund_return_20d",
        "fund_volatility_20d", "fund_drawdown_20d", "mean_reversion",
        "future_return_pct_next_day", "future_return_pct_1w",
        "label_1d", "label_1w",
    ]
    rename_map = {
        "return_1d": "fund_return_1d",
        "return_5d": "fund_return_5d",
        "return_20d": "fund_return_20d",
        "volatility_20d": "fund_volatility_20d",
        "drawdown_20d": "fund_drawdown_20d",
    }
    for old, new in rename_map.items():
        if old in df.columns and new not in df.columns:
            df = df.rename(columns={old: new})

    available_cols = [c for c in output_cols if c in df.columns]
    return df[available_cols]


if __name__ == "__main__":
    main()
