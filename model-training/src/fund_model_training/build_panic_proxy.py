from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np

from fund_model_training.build_panic_factor import DEFAULT_WEIGHTS, build_panic_factor_from_frame
from fund_model_training.collectors.common import require_pandas, write_csv


def main() -> None:
    parser = argparse.ArgumentParser(description="Build a public-data panic proxy from index and futures daily CSVs.")
    parser.add_argument("--index-daily", type=Path, required=True)
    parser.add_argument("--futures", type=Path)
    parser.add_argument("--market", default="CN")
    parser.add_argument("--components-output", type=Path, help="Optional component CSV output.")
    parser.add_argument("--output", type=Path, required=True, help="panic_factor CSV output.")
    args = parser.parse_args()

    components = build_panic_proxy_components(args.index_daily, args.futures, market=args.market)
    if args.components_output:
        write_csv(components, args.components_output)
    panic = build_panic_factor_from_frame(
        raw=components,
        market=args.market,
        timestamp_col="timestamp",
        available_time_col="available_time",
        weights=DEFAULT_WEIGHTS,
    )
    out = write_csv(panic, args.output)
    print(json.dumps({
        "ok": True,
        "component_rows": int(len(components)),
        "panic_rows": int(len(panic)),
        "output": str(out),
    }, ensure_ascii=False, indent=2))


def build_panic_proxy_components(index_daily_path: str | Path, futures_path: str | Path | None = None, market: str = "CN"):
    pd = require_pandas()
    index = pd.read_csv(index_daily_path, dtype={"index_code": str})
    index["trade_date"] = pd.to_datetime(index["trade_date"], errors="coerce")
    index["close"] = pd.to_numeric(index["close"], errors="coerce")
    index = index.dropna(subset=["trade_date", "close"]).sort_values(["index_code", "trade_date"])
    group = index.groupby("index_code", group_keys=False)
    index["index_return_1d"] = group["close"].pct_change() * 100.0
    index["index_volatility_20d"] = group["index_return_1d"].transform(lambda s: s.rolling(20, min_periods=3).std())
    index["index_high_20d"] = group["close"].transform(lambda s: s.rolling(20, min_periods=3).max())
    index["drawdown_20d"] = index["close"] / index["index_high_20d"] * 100.0 - 100.0

    daily = index.groupby("trade_date", as_index=False).agg({
        "index_return_1d": "mean",
        "index_volatility_20d": "mean",
        "drawdown_20d": "mean",
    })

    if futures_path and Path(futures_path).exists():
        futures = pd.read_csv(futures_path)
        futures["timestamp"] = pd.to_datetime(futures["timestamp"], errors="coerce")
        futures["trade_date"] = futures["timestamp"].dt.normalize()
        futures["price"] = pd.to_numeric(futures["price"], errors="coerce")
        futures["open_interest"] = pd.to_numeric(futures.get("open_interest"), errors="coerce")
        futures = futures.dropna(subset=["trade_date", "price"]).sort_values(["contract", "trade_date"])
        f_group = futures.groupby("contract", group_keys=False)
        futures["futures_return_1d"] = f_group["price"].pct_change() * 100.0
        futures["open_interest_change_5d"] = f_group["open_interest"].pct_change(5) * 100.0
        futures_daily = futures.groupby("trade_date", as_index=False).agg({
            "futures_return_1d": "mean",
            "open_interest_change_5d": "mean",
        })
        daily = daily.merge(futures_daily, on="trade_date", how="left")
    else:
        daily["futures_return_1d"] = 0.0
        daily["open_interest_change_5d"] = 0.0

    negative_index = (-pd.to_numeric(daily["index_return_1d"], errors="coerce")).clip(lower=0)
    negative_futures = (-pd.to_numeric(daily["futures_return_1d"], errors="coerce")).clip(lower=0)
    volatility = pd.to_numeric(daily["index_volatility_20d"], errors="coerce").fillna(0.0)
    drawdown = (-pd.to_numeric(daily["drawdown_20d"], errors="coerce")).clip(lower=0)
    open_interest_change = pd.to_numeric(daily["open_interest_change_5d"], errors="coerce").abs().fillna(0.0)

    out = pd.DataFrame({
        "market": market,
        "timestamp": daily["trade_date"].dt.strftime("%Y-%m-%d 16:00:00"),
        "available_time": daily["trade_date"].dt.strftime("%Y-%m-%d 16:00:00"),
        "iv_component": volatility + negative_futures * 0.25,
        "flow_component": open_interest_change + negative_index * 0.25,
        "news_component": negative_index + drawdown * 0.10,
        "limit_component": drawdown + negative_index * 0.50,
    })
    return out.replace([np.inf, -np.inf], np.nan).fillna(0.0)


if __name__ == "__main__":
    main()
