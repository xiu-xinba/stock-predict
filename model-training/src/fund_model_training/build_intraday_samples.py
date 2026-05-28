from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np

from fund_model_training.collectors.common import require_pandas, write_csv
from fund_model_training.labels import labels_from_future_return


def main() -> None:
    parser = argparse.ArgumentParser(description="Build processed 3/5-minute index-fund training samples.")
    parser.add_argument("--fund-intraday", type=Path, required=True, help="fund_intraday contract CSV.")
    parser.add_argument("--index-intraday", type=Path, required=True, help="index_intraday contract CSV.")
    parser.add_argument("--output", type=Path, required=True, help="Output processed sample CSV.")
    parser.add_argument("--dim-fund", type=Path, help="dim_fund contract CSV.")
    parser.add_argument("--panic-factor", type=Path, help="panic_factor contract CSV.")
    parser.add_argument("--fund-code", help="Filter to one fund code.")
    parser.add_argument("--tracking-index", help="Tracking index code for the sample rows.")
    parser.add_argument("--market", default="CN")
    parser.add_argument("--horizon-minutes", type=int, default=5, choices=[3, 5])
    parser.add_argument("--flat-threshold-pct", type=float, default=0.02)
    args = parser.parse_args()

    df = build_intraday_samples(
        fund_intraday_path=args.fund_intraday,
        index_intraday_path=args.index_intraday,
        dim_fund_path=args.dim_fund,
        panic_factor_path=args.panic_factor,
        fund_code=args.fund_code,
        tracking_index=args.tracking_index,
        market=args.market,
        horizon_minutes=args.horizon_minutes,
        flat_threshold_pct=args.flat_threshold_pct,
    )
    out = write_csv(df, args.output)
    print(json.dumps({"ok": True, "rows": int(len(df)), "output": str(out)}, ensure_ascii=False, indent=2))


def build_intraday_samples(
    fund_intraday_path: str | Path,
    index_intraday_path: str | Path,
    dim_fund_path: str | Path | None = None,
    panic_factor_path: str | Path | None = None,
    fund_code: str | None = None,
    tracking_index: str | None = None,
    market: str = "CN",
    horizon_minutes: int = 5,
    flat_threshold_pct: float = 0.02,
):
    pd = require_pandas()
    fund = pd.read_csv(fund_intraday_path, dtype={"fund_code": str})
    fund["fund_code"] = fund["fund_code"].astype(str).str.zfill(6)
    fund["timestamp"] = pd.to_datetime(fund["timestamp"], errors="coerce")
    fund["available_time"] = pd.to_datetime(fund.get("available_time", fund["timestamp"]), errors="coerce")
    fund["price"] = pd.to_numeric(fund["price"], errors="coerce")
    fund = fund.dropna(subset=["timestamp", "available_time", "price"]).sort_values(["fund_code", "timestamp"])
    if fund_code:
        fund = fund.loc[fund["fund_code"] == str(fund_code).zfill(6)].copy()
    if fund.empty:
        raise ValueError("No fund_intraday rows remain after filtering.")

    fund = _attach_intraday_metadata(fund, dim_fund_path, tracking_index=tracking_index, market=market)
    fund = _add_fund_intraday_features(fund)
    fund = _add_future_return(fund, horizon_minutes)

    out = _join_index_intraday_features(
        fund,
        index_intraday_path,
        tracking_index=tracking_index,
        horizon_minutes=horizon_minutes,
    )
    if panic_factor_path:
        out = _join_panic_intraday_features(out, panic_factor_path)
    else:
        _ensure_empty_panic_features(out)

    horizon_col = f"future_return_pct_{horizon_minutes}m"
    out["label"] = labels_from_future_return(out[horizon_col], flat_threshold_pct)
    out["asof_time"] = out["available_time"]
    out["sample_timestamp"] = out["timestamp"]
    out["change_pct"] = out["fund_return_1m"]
    out["index_change_pct"] = out["index_return_1m"]
    out["market_change_pct"] = out["index_return_1m"]
    index_future_col = f"future_index_return_pct_{horizon_minutes}m"
    if index_future_col in out.columns:
        out[f"future_tracking_error_pct_{horizon_minutes}m"] = out[horizon_col] - out[index_future_col]
    out["intraday_liquidity"] = out["fund_volume_ratio_20m"] - out["bid_ask_spread_pct"] * 0.2
    out["etf_flow_proxy"] = out["fund_volume_ratio_20m"] * 0.5 + out["premium_pct"] * 0.5
    out = out.replace([np.inf, -np.inf], np.nan)
    out = out.dropna(subset=[horizon_col, "label"]).reset_index(drop=True)
    return _order_intraday_columns(out, horizon_col)


def _attach_intraday_metadata(fund, dim_fund_path, tracking_index: str | None, market: str):
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
        out["tracking_index"] = out["tracking_index"].fillna(tracking_index or "UNMAPPED").replace("", tracking_index or "UNMAPPED")
    if "market" not in out.columns:
        out["market"] = market
    else:
        out["market"] = out["market"].fillna(market).replace("", market)
    return out


def _add_fund_intraday_features(fund):
    group = fund.groupby("fund_code", group_keys=False)
    fund["fund_return_1m"] = group["price"].pct_change() * 100.0
    fund["fund_return_3m"] = group["price"].pct_change(3) * 100.0
    fund["fund_return_5m"] = group["price"].pct_change(5) * 100.0
    fund["fund_volatility_15m"] = group["fund_return_1m"].transform(lambda s: s.rolling(15, min_periods=3).std())
    if "volume" in fund.columns:
        volume = pd_to_numeric(fund["volume"])
        fund["fund_volume_ratio_20m"] = (
            volume / volume.groupby(fund["fund_code"]).transform(lambda s: s.rolling(20, min_periods=3).mean()).replace(0, np.nan)
        )
    else:
        fund["fund_volume_ratio_20m"] = 1.0
    fund["premium_pct"] = pd_to_numeric(fund.get("premium_pct", 0.0), fund.index).fillna(0.0)
    spread = pd_to_numeric(fund.get("bid_ask_spread", 0.0), fund.index).fillna(0.0)
    fund["bid_ask_spread_pct"] = (spread / fund["price"].replace(0, np.nan) * 100.0).fillna(0.0)
    return fund


def _add_future_return(fund, horizon_minutes: int):
    pd = require_pandas()
    future_col = f"future_return_pct_{horizon_minutes}m"
    pieces = []
    for _, group in fund.groupby("fund_code", group_keys=False):
        left = group.sort_values("timestamp").copy()
        left["target_time"] = left["timestamp"] + pd.Timedelta(minutes=horizon_minutes)
        right = left[["timestamp", "price"]].rename(columns={"timestamp": "future_timestamp", "price": "future_price"})
        merged = pd.merge_asof(
            left.sort_values("target_time"),
            right.sort_values("future_timestamp"),
            left_on="target_time",
            right_on="future_timestamp",
            direction="forward",
            tolerance=pd.Timedelta(minutes=max(1, horizon_minutes)),
        ).sort_values("timestamp")
        merged[future_col] = merged["future_price"] / merged["price"] * 100.0 - 100.0
        pieces.append(merged)
    return pd.concat(pieces, ignore_index=True)


def _join_index_intraday_features(samples, index_intraday_path, tracking_index: str | None, horizon_minutes: int):
    pd = require_pandas()
    index = pd.read_csv(index_intraday_path, dtype={"index_code": str})
    index["timestamp"] = pd.to_datetime(index["timestamp"], errors="coerce")
    index["price"] = pd.to_numeric(index["price"], errors="coerce")
    index = index.dropna(subset=["timestamp", "price"]).sort_values(["index_code", "timestamp"])
    if tracking_index:
        index = index.loc[index["index_code"] == str(tracking_index)].copy()
    elif "tracking_index" in samples.columns:
        sample_indexes = [
            str(value)
            for value in samples["tracking_index"].dropna().unique()
            if str(value) and str(value) != "UNMAPPED"
        ]
        if len(sample_indexes) == 1:
            index = index.loc[index["index_code"] == sample_indexes[0]].copy()
    if index.empty:
        return _ensure_empty_index_intraday_features(samples.copy())
    group = index.groupby("index_code", group_keys=False)
    index["index_return_1m"] = group["price"].pct_change() * 100.0
    index["index_return_3m"] = group["price"].pct_change(3) * 100.0
    index["index_return_5m"] = group["price"].pct_change(5) * 100.0
    index["index_volatility_15m"] = group["index_return_1m"].transform(lambda s: s.rolling(15, min_periods=3).std())
    keep = ["timestamp", "index_code", "price", "index_return_1m", "index_return_3m", "index_return_5m", "index_volatility_15m"]
    right = index[keep].rename(columns={"timestamp": "index_timestamp", "price": "index_price"})
    out = pd.merge_asof(
        samples.sort_values("timestamp"),
        right.sort_values("index_timestamp"),
        left_on="timestamp",
        right_on="index_timestamp",
        direction="backward",
        tolerance=pd.Timedelta(minutes=2),
    )
    future_right = index[["timestamp", "price"]].rename(
        columns={"timestamp": "future_index_timestamp", "price": "future_index_price"}
    )
    out = pd.merge_asof(
        out.sort_values("target_time"),
        future_right.sort_values("future_index_timestamp"),
        left_on="target_time",
        right_on="future_index_timestamp",
        direction="forward",
        tolerance=pd.Timedelta(minutes=max(1, horizon_minutes)),
    ).sort_values("timestamp")
    out[f"future_index_return_pct_{horizon_minutes}m"] = (
        out["future_index_price"] / out["index_price"].replace(0, np.nan) * 100.0 - 100.0
    )
    out["fund_index_spread_1m"] = out["fund_return_1m"] - out["index_return_1m"]
    out["fund_index_spread_5m"] = out["fund_return_5m"] - out["index_return_5m"]
    return _ensure_empty_index_intraday_features(out)


def _join_panic_intraday_features(samples, panic_factor_path):
    pd = require_pandas()
    panic = pd.read_csv(panic_factor_path)
    panic["timestamp"] = pd.to_datetime(panic["timestamp"], errors="coerce")
    panic = panic.rename(columns={
        "iv_component": "panic_iv_component",
        "flow_component": "panic_flow_component",
        "news_component": "panic_news_component",
        "limit_component": "panic_limit_component",
    }).dropna(subset=["timestamp"]).sort_values("timestamp")
    columns = [
        "timestamp",
        "market",
        "fear_score",
        "panic_iv_component",
        "panic_flow_component",
        "panic_news_component",
        "panic_limit_component",
    ]
    if "market" in panic.columns and "market" in samples.columns:
        pieces = []
        for market, group in samples.groupby("market", group_keys=False):
            market_panic = panic.loc[panic["market"].astype(str).str.upper() == str(market).upper(), columns].copy()
            if market_panic.empty:
                pieces.append(_ensure_empty_panic_features(group.copy()))
                continue
            joined = pd.merge_asof(
                group.sort_values("timestamp"),
                market_panic.sort_values("timestamp").rename(columns={"timestamp": "panic_timestamp"}),
                left_on="timestamp",
                right_on="panic_timestamp",
                direction="backward",
            )
            if "market_x" in joined.columns:
                joined = joined.rename(columns={"market_x": "market"}).drop(columns=[c for c in ("market_y",) if c in joined.columns])
            pieces.append(_ensure_empty_panic_features(joined))
        return pd.concat(pieces, ignore_index=True)
    joined = pd.merge_asof(
        samples.sort_values("timestamp"),
        panic[columns].sort_values("timestamp").rename(columns={"timestamp": "panic_timestamp"}),
        left_on="timestamp",
        right_on="panic_timestamp",
        direction="backward",
    )
    return _ensure_empty_panic_features(joined)


def _ensure_empty_index_intraday_features(df):
    for name in (
        "index_price",
        "index_return_1m",
        "index_return_3m",
        "index_return_5m",
        "index_volatility_15m",
        "fund_index_spread_1m",
        "fund_index_spread_5m",
    ):
        if name not in df.columns:
            df[name] = 0.0
        df[name] = pd_to_numeric(df[name], df.index).fillna(0.0)
    return df


def _ensure_empty_panic_features(df):
    for name in ("fear_score", "panic_iv_component", "panic_flow_component", "panic_news_component", "panic_limit_component"):
        if name not in df.columns:
            df[name] = 0.0
        df[name] = pd_to_numeric(df[name], df.index).fillna(0.0)
    return df


def _order_intraday_columns(df, horizon_col: str):
    preferred = [
        "fund_code",
        "fund_name",
        "fund_type",
        "market",
        "tracking_index",
        "sample_timestamp",
        "asof_time",
        "target_time",
        horizon_col,
        horizon_col.replace("future_return_pct_", "future_index_return_pct_"),
        horizon_col.replace("future_return_pct_", "future_tracking_error_pct_"),
        "label",
        "price",
        "future_price",
        "future_index_price",
        "fund_return_1m",
        "fund_return_3m",
        "fund_return_5m",
        "fund_volatility_15m",
        "fund_volume_ratio_20m",
        "premium_pct",
        "bid_ask_spread_pct",
        "index_price",
        "index_return_1m",
        "index_return_3m",
        "index_return_5m",
        "index_volatility_15m",
        "fund_index_spread_1m",
        "fund_index_spread_5m",
        "intraday_liquidity",
        "etf_flow_proxy",
        "fear_score",
        "panic_iv_component",
        "panic_flow_component",
        "panic_news_component",
        "panic_limit_component",
        "change_pct",
        "index_change_pct",
        "market_change_pct",
    ]
    ordered = [column for column in preferred if column in df.columns]
    rest = [column for column in df.columns if column not in ordered]
    return df[ordered + rest]


def pd_to_numeric(values, index=None):
    pd = require_pandas()
    if not hasattr(values, "index"):
        return pd.Series(values, index=index)
    return pd.to_numeric(values, errors="coerce")


if __name__ == "__main__":
    main()
