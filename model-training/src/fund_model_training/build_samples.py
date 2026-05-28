from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np

from fund_model_training.collectors.common import require_pandas, write_csv
from fund_model_training.labels import labels_from_future_return


def main() -> None:
    parser = argparse.ArgumentParser(description="Build processed index-fund daily/weekly training samples.")
    parser.add_argument("--fund-daily", type=Path, required=True, help="fund_daily contract CSV.")
    parser.add_argument("--output", type=Path, required=True, help="Output processed sample CSV.")
    parser.add_argument("--dim-fund", type=Path, help="dim_fund contract CSV.")
    parser.add_argument("--index-daily", type=Path, help="index_daily contract CSV.")
    parser.add_argument("--futures", type=Path, help="futures_bar contract CSV.")
    parser.add_argument("--futures-underlying", help="Filter futures rows to this underlying before joining.")
    parser.add_argument("--panic-factor", type=Path, help="panic_factor contract CSV.")
    parser.add_argument("--fund-code", help="Filter to one fund code.")
    parser.add_argument("--tracking-index", help="Override or fill tracking_index for the samples.")
    parser.add_argument("--market", default="CN", help="Override/fill market when missing.")
    parser.add_argument("--flat-threshold-pct", type=float, default=0.05)
    parser.add_argument("--weekly-periods", type=int, default=5)
    args = parser.parse_args()

    df = build_daily_weekly_samples(
        fund_daily_path=args.fund_daily,
        dim_fund_path=args.dim_fund,
        index_daily_path=args.index_daily,
        futures_path=args.futures,
        futures_underlying=args.futures_underlying,
        panic_factor_path=args.panic_factor,
        fund_code=args.fund_code,
        tracking_index=args.tracking_index,
        market=args.market,
        flat_threshold_pct=args.flat_threshold_pct,
        weekly_periods=args.weekly_periods,
    )
    out = write_csv(df, args.output)
    print(json.dumps({"ok": True, "rows": int(len(df)), "output": str(out)}, ensure_ascii=False, indent=2))


def build_daily_weekly_samples(
    fund_daily_path: str | Path,
    output_feature_compat: bool = True,
    dim_fund_path: str | Path | None = None,
    index_daily_path: str | Path | None = None,
    futures_path: str | Path | None = None,
    futures_underlying: str | None = None,
    panic_factor_path: str | Path | None = None,
    fund_code: str | None = None,
    tracking_index: str | None = None,
    market: str = "CN",
    flat_threshold_pct: float = 0.05,
    weekly_periods: int = 5,
):
    pd = require_pandas()
    fund = pd.read_csv(fund_daily_path, dtype={"fund_code": str})
    fund["fund_code"] = fund["fund_code"].astype(str).str.zfill(6)
    fund["trade_date"] = pd.to_datetime(fund["trade_date"], errors="coerce")
    fund["available_time"] = pd.to_datetime(fund["available_time"], errors="coerce")
    fund = fund.dropna(subset=["trade_date", "available_time"]).sort_values(["fund_code", "trade_date"])
    if fund_code:
        fund = fund.loc[fund["fund_code"] == str(fund_code).zfill(6)].copy()
    if fund.empty:
        raise ValueError("No fund_daily rows remain after filtering.")

    fund = _attach_fund_metadata(fund, dim_fund_path, tracking_index=tracking_index, market=market)
    value_col = _first_existing(fund.columns, ("adjusted_nav", "nav"))
    fund["latest_nav"] = pd.to_numeric(fund[value_col], errors="coerce")
    group = fund.groupby("fund_code", group_keys=False)
    fund["fund_return_1d"] = group["latest_nav"].pct_change() * 100.0
    fund["fund_return_5d"] = group["latest_nav"].pct_change(5) * 100.0
    fund["fund_volatility_20d"] = group["fund_return_1d"].transform(lambda s: s.rolling(20, min_periods=3).std())
    fund["future_return_pct_next_day"] = group["latest_nav"].shift(-1) / fund["latest_nav"] * 100.0 - 100.0
    fund["future_return_pct_1w"] = group["latest_nav"].shift(-weekly_periods) / fund["latest_nav"] * 100.0 - 100.0
    fund["label"] = labels_from_future_return(fund["future_return_pct_next_day"], flat_threshold_pct)
    fund["label_1d"] = fund["label"]
    fund["label_1w"] = labels_from_future_return(fund["future_return_pct_1w"], flat_threshold_pct)

    out = fund.rename(columns={"trade_date": "sample_trade_date"}).copy()
    out["asof_time"] = out["available_time"]
    out["nav"] = out["latest_nav"]

    if index_daily_path:
        out = _join_index_features(out, index_daily_path, weekly_periods=weekly_periods)
    else:
        _ensure_empty_index_features(out)

    if futures_path:
        out = _join_futures_features(out, futures_path, futures_underlying=futures_underlying)
    else:
        _ensure_empty_futures_features(out)

    if panic_factor_path:
        out = _join_panic_features(out, panic_factor_path)
    else:
        _ensure_empty_panic_features(out)

    out["fund_tracking_error_1d"] = out["fund_return_1d"] - out["index_return_1d"]
    out["fund_tracking_error_5d"] = out["fund_return_5d"] - out["index_return_5d"]
    if "future_index_return_pct_next_day" in out.columns:
        out["future_tracking_error_pct_next_day"] = out["future_return_pct_next_day"] - out["future_index_return_pct_next_day"]
    if "future_index_return_pct_1w" in out.columns:
        out["future_tracking_error_pct_1w"] = out["future_return_pct_1w"] - out["future_index_return_pct_1w"]
    out["market_change_pct"] = out["index_return_1d"]
    out["index_change_pct"] = out["index_return_1d"]
    out["change_pct"] = out["fund_return_1d"]
    out["volume_ratio"] = out.get("volume_ratio", 1.0)
    out["fund_flow_pct"] = out.get("fund_flow_pct", 0.0)

    if output_feature_compat:
        out = _order_sample_columns(out)
    out = out.replace([np.inf, -np.inf], np.nan)
    return out.dropna(subset=["future_return_pct_next_day", "label"]).reset_index(drop=True)


def _attach_fund_metadata(fund, dim_fund_path, tracking_index: str | None, market: str):
    pd = require_pandas()
    out = fund.copy()
    if dim_fund_path:
        dim = pd.read_csv(dim_fund_path, dtype={"fund_code": str})
        dim["fund_code"] = dim["fund_code"].astype(str).str.zfill(6)
        columns = [c for c in ("fund_code", "fund_name", "fund_type", "tracking_index", "market") if c in dim.columns]
        out = out.merge(dim[columns], on="fund_code", how="left")
    if "fund_name" not in out.columns:
        out["fund_name"] = out["fund_code"]
    if "fund_type" not in out.columns:
        out["fund_type"] = "INDEX_FUND"
    if "tracking_index" not in out.columns:
        out["tracking_index"] = tracking_index or "UNMAPPED"
    else:
        out["tracking_index"] = out["tracking_index"].fillna(tracking_index or "UNMAPPED")
        if tracking_index:
            out.loc[out["tracking_index"].eq("UNMAPPED") | out["tracking_index"].eq(""), "tracking_index"] = tracking_index
    if "market" not in out.columns:
        out["market"] = market
    else:
        out["market"] = out["market"].fillna(market).replace("", market)
    return out


def _join_index_features(samples, index_daily_path, weekly_periods: int):
    pd = require_pandas()
    index = pd.read_csv(index_daily_path, dtype={"index_code": str})
    index["trade_date"] = pd.to_datetime(index["trade_date"], errors="coerce")
    index["index_close"] = pd.to_numeric(index["close"], errors="coerce")
    index = index.dropna(subset=["trade_date", "index_close"]).sort_values(["index_code", "trade_date"])
    group = index.groupby("index_code", group_keys=False)
    index["index_return_1d"] = group["index_close"].pct_change() * 100.0
    index["index_return_5d"] = group["index_close"].pct_change(5) * 100.0
    index["index_volatility_20d"] = group["index_return_1d"].transform(lambda s: s.rolling(20, min_periods=3).std())
    index["index_high_20d"] = group["index_close"].transform(lambda s: s.rolling(20, min_periods=3).max())
    index["index_drawdown_20d"] = index["index_close"] / index["index_high_20d"] * 100.0 - 100.0
    index["future_index_return_pct_next_day"] = group["index_close"].shift(-1) / index["index_close"] * 100.0 - 100.0
    index["future_index_return_pct_1w"] = group["index_close"].shift(-weekly_periods) / index["index_close"] * 100.0 - 100.0
    keep = [
        "index_code",
        "trade_date",
        "index_close",
        "index_return_1d",
        "index_return_5d",
        "index_volatility_20d",
        "index_drawdown_20d",
        "future_index_return_pct_next_day",
        "future_index_return_pct_1w",
    ]
    out = samples.merge(
        index[keep],
        left_on=["tracking_index", "sample_trade_date"],
        right_on=["index_code", "trade_date"],
        how="left",
    )
    out = out.drop(columns=[c for c in ("index_code", "trade_date") if c in out.columns])
    _ensure_empty_index_features(out)
    return out


def _join_futures_features(samples, futures_path, futures_underlying: str | None = None):
    pd = require_pandas()
    futures = pd.read_csv(futures_path)
    futures["timestamp"] = pd.to_datetime(futures["timestamp"], errors="coerce")
    futures["trade_date"] = futures["timestamp"].dt.normalize()
    futures["price"] = pd.to_numeric(futures["price"], errors="coerce")
    futures["open_interest"] = pd.to_numeric(futures.get("open_interest"), errors="coerce")
    futures["basis"] = pd.to_numeric(futures.get("basis"), errors="coerce")
    if futures_underlying and "underlying" in futures.columns:
        futures = futures.loc[futures["underlying"].astype(str).str.upper() == futures_underlying.upper()].copy()
    futures = futures.dropna(subset=["trade_date", "price"]).sort_values(["contract", "trade_date"])
    if futures.empty:
        _ensure_empty_futures_features(samples)
        return samples
    group = futures.groupby("contract", group_keys=False)
    futures["futures_return_1d"] = group["price"].pct_change() * 100.0
    futures["futures_open_interest_change_5d"] = group["open_interest"].pct_change(5) * 100.0
    daily = futures.groupby("trade_date", as_index=False).agg({
        "futures_return_1d": "mean",
        "basis": "mean",
        "futures_open_interest_change_5d": "mean",
    }).rename(columns={"basis": "futures_basis"})
    out = samples.merge(daily, left_on="sample_trade_date", right_on="trade_date", how="left")
    out = out.drop(columns=[c for c in ("trade_date",) if c in out.columns])
    _ensure_empty_futures_features(out)
    return out


def _join_panic_features(samples, panic_factor_path):
    pd = require_pandas()
    panic = pd.read_csv(panic_factor_path)
    panic["timestamp"] = pd.to_datetime(panic["timestamp"], errors="coerce")
    panic["trade_date"] = panic["timestamp"].dt.normalize()
    rename = {
        "iv_component": "panic_iv_component",
        "flow_component": "panic_flow_component",
        "news_component": "panic_news_component",
        "limit_component": "panic_limit_component",
    }
    panic = panic.rename(columns=rename)
    columns = [
        "market",
        "trade_date",
        "fear_score",
        "panic_iv_component",
        "panic_flow_component",
        "panic_news_component",
        "panic_limit_component",
    ]
    out = samples.merge(panic[columns], left_on=["market", "sample_trade_date"], right_on=["market", "trade_date"], how="left")
    out = out.drop(columns=[c for c in ("trade_date",) if c in out.columns])
    _ensure_empty_panic_features(out)
    return out


def _ensure_empty_index_features(df) -> None:
    for name in ("index_close", "index_return_1d", "index_return_5d", "index_volatility_20d", "index_drawdown_20d"):
        if name not in df.columns:
            df[name] = 0.0
        df[name] = df[name].fillna(0.0)


def _ensure_empty_futures_features(df) -> None:
    for name in ("futures_return_1d", "futures_basis", "futures_open_interest_change_5d"):
        if name not in df.columns:
            df[name] = 0.0
        df[name] = df[name].fillna(0.0)


def _ensure_empty_panic_features(df) -> None:
    for name in ("fear_score", "panic_iv_component", "panic_flow_component", "panic_news_component", "panic_limit_component"):
        if name not in df.columns:
            df[name] = 0.0
        df[name] = df[name].fillna(0.0)


def _first_existing(columns, candidates):
    present = set(columns)
    for candidate in candidates:
        if candidate in present:
            return candidate
    raise ValueError(f"Missing any of columns: {', '.join(candidates)}")


def _order_sample_columns(df):
    preferred = [
        "fund_code",
        "fund_name",
        "fund_type",
        "market",
        "tracking_index",
        "sample_trade_date",
        "asof_time",
        "latest_nav",
        "nav",
        "future_return_pct_next_day",
        "future_return_pct_1w",
        "label",
        "label_1d",
        "label_1w",
        "fund_return_1d",
        "fund_return_5d",
        "fund_volatility_20d",
        "index_close",
        "index_return_1d",
        "index_return_5d",
        "index_volatility_20d",
        "index_drawdown_20d",
        "future_index_return_pct_next_day",
        "future_index_return_pct_1w",
        "future_tracking_error_pct_next_day",
        "future_tracking_error_pct_1w",
        "fund_tracking_error_1d",
        "fund_tracking_error_5d",
        "futures_return_1d",
        "futures_basis",
        "futures_open_interest_change_5d",
        "fear_score",
        "panic_iv_component",
        "panic_flow_component",
        "panic_news_component",
        "panic_limit_component",
        "market_change_pct",
        "index_change_pct",
        "change_pct",
        "volume_ratio",
        "fund_flow_pct",
    ]
    ordered = [column for column in preferred if column in df.columns]
    rest = [column for column in df.columns if column not in ordered]
    return df[ordered + rest]


if __name__ == "__main__":
    main()
