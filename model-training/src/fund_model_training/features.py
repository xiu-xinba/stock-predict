from __future__ import annotations

import numpy as np
import pandas as pd

from .schema import REQUIRED_COLUMNS, get_feature_names


def prepare_features(df: pd.DataFrame, feature_set: str) -> tuple[pd.DataFrame, list[str]]:
    feature_names = get_feature_names(feature_set)
    out = df.copy()
    _validate_base_columns(out)
    out["asof_time"] = pd.to_datetime(out["asof_time"], errors="coerce")
    out = out.dropna(subset=["asof_time"]).sort_values(["fund_code", "asof_time"])

    value_col = _first_existing(out, ["estimated_nav", "latest_nav", "nav", "price", "close"])
    if value_col:
        returns = out.groupby("fund_code", group_keys=False)[value_col].pct_change() * 100.0
    else:
        returns = pd.Series(0.0, index=out.index)

    _ensure_feature(out, "momentum_5d", lambda: _group_pct_change(out, value_col, 5))
    _ensure_feature(out, "return_10d", lambda: _group_pct_change(out, value_col, 10))
    _ensure_feature(out, "volatility_20d", lambda: _rolling_std(returns, out["fund_code"], 20))
    _ensure_feature(out, "volume_ratio", lambda: _volume_ratio(out))
    _ensure_feature(out, "market_beta", lambda: _market_beta(returns, out))
    _ensure_feature(out, "sector_momentum", lambda: _coalesce(out, ["sector_change_pct", "sector_return_pct", "industry_change_pct"]))
    _ensure_feature(out, "flow_signal", lambda: _flow_signal(out))
    _ensure_feature(out, "mean_reversion", lambda: -_rolling_mean(returns, out["fund_code"], 5))
    _ensure_feature(out, "holding_exposure", lambda: _holding_exposure(out))
    _ensure_feature(out, "intraday_liquidity", lambda: _intraday_liquidity(out))
    _ensure_feature(out, "etf_flow_proxy", lambda: _etf_flow_proxy(out))
    _ensure_feature(out, "fund_return_1d", lambda: returns)
    _ensure_feature(out, "fund_return_5d", lambda: _group_pct_change(out, value_col, 5))
    _ensure_feature(out, "fund_volatility_20d", lambda: _rolling_std(returns, out["fund_code"], 20))
    _ensure_feature(out, "index_return_1d", lambda: _coalesce(out, ["index_return_1d", "index_change_pct", "market_change_pct"]))
    _ensure_feature(out, "index_return_5d", lambda: _coalesce(out, ["index_return_5d", "market_return_5d"]))
    _ensure_feature(out, "index_volatility_20d", lambda: _coalesce(out, ["index_volatility_20d", "market_volatility_20d"]))
    _ensure_feature(out, "fund_tracking_error_1d", lambda: out["fund_return_1d"] - out["index_return_1d"])
    _ensure_feature(out, "fund_tracking_error_5d", lambda: out["fund_return_5d"] - out["index_return_5d"])
    _ensure_feature(out, "futures_return_1d", lambda: _coalesce(out, ["futures_return_1d", "index_futures_return_1d"]))
    _ensure_feature(out, "futures_basis", lambda: _coalesce(out, ["futures_basis", "index_futures_basis"]))
    _ensure_feature(out, "futures_open_interest_change_5d", lambda: _coalesce(out, ["futures_open_interest_change_5d"]))
    _ensure_feature(out, "fear_score", lambda: _coalesce(out, ["fear_score", "panic_score"]))
    _ensure_feature(out, "panic_iv_component", lambda: _coalesce(out, ["panic_iv_component", "iv_component"]))
    _ensure_feature(out, "panic_flow_component", lambda: _coalesce(out, ["panic_flow_component", "flow_component"]))
    _ensure_feature(out, "panic_news_component", lambda: _coalesce(out, ["panic_news_component", "news_component"]))
    _ensure_feature(out, "panic_limit_component", lambda: _coalesce(out, ["panic_limit_component", "limit_component"]))
    _ensure_feature(out, "fund_return_1m", lambda: _group_pct_change(out, value_col, 1))
    _ensure_feature(out, "fund_return_3m", lambda: _group_pct_change(out, value_col, 3))
    _ensure_feature(out, "fund_return_5m", lambda: _group_pct_change(out, value_col, 5))
    _ensure_feature(out, "fund_volatility_15m", lambda: _rolling_std(out["fund_return_1m"], out["fund_code"], 15))
    _ensure_feature(out, "fund_volume_ratio_20m", lambda: _volume_ratio_window(out, 20))
    _ensure_feature(out, "premium_pct", lambda: _coalesce(out, ["premium_pct", "nav_premium_pct"]))
    _ensure_feature(out, "bid_ask_spread_pct", lambda: _bid_ask_spread_pct(out, value_col))
    _ensure_feature(out, "index_return_1m", lambda: _coalesce(out, ["index_return_1m", "index_change_pct"]))
    _ensure_feature(out, "index_return_3m", lambda: _coalesce(out, ["index_return_3m"]))
    _ensure_feature(out, "index_return_5m", lambda: _coalesce(out, ["index_return_5m", "market_return_5m"]))
    _ensure_feature(out, "index_volatility_15m", lambda: _coalesce(out, ["index_volatility_15m"]))
    _ensure_feature(out, "fund_index_spread_1m", lambda: out["fund_return_1m"] - out["index_return_1m"])
    _ensure_feature(out, "fund_index_spread_5m", lambda: out["fund_return_5m"] - out["index_return_5m"])

    for name in feature_names:
        out[name] = pd.to_numeric(out[name], errors="coerce")

    out[feature_names] = out[feature_names].replace([np.inf, -np.inf], np.nan).fillna(0.0)
    return out, feature_names


def _validate_base_columns(df: pd.DataFrame) -> None:
    missing = [c for c in REQUIRED_COLUMNS if c not in df.columns]
    if missing:
        raise ValueError(f"Missing required columns: {', '.join(missing)}")


def _ensure_feature(df: pd.DataFrame, name: str, factory) -> None:
    if name not in df.columns:
        df[name] = factory()


def _first_existing(df: pd.DataFrame, names: list[str]) -> str | None:
    return next((name for name in names if name in df.columns), None)


def _coalesce(df: pd.DataFrame, names: list[str]) -> pd.Series:
    result = pd.Series(np.nan, index=df.index, dtype="float64")
    for name in names:
        if name in df.columns:
            result = result.fillna(pd.to_numeric(df[name], errors="coerce"))
    return result.fillna(0.0)


def _group_pct_change(df: pd.DataFrame, value_col: str | None, periods: int) -> pd.Series:
    if not value_col:
        return pd.Series(0.0, index=df.index)
    return df.groupby("fund_code", group_keys=False)[value_col].pct_change(periods=periods) * 100.0


def _rolling_std(values: pd.Series, groups: pd.Series, window: int) -> pd.Series:
    return values.groupby(groups).transform(lambda s: s.rolling(window, min_periods=3).std())


def _rolling_mean(values: pd.Series, groups: pd.Series, window: int) -> pd.Series:
    return values.groupby(groups).transform(lambda s: s.rolling(window, min_periods=2).mean())


def _volume_ratio(df: pd.DataFrame) -> pd.Series:
    if "volume_ratio" in df.columns:
        return pd.to_numeric(df["volume_ratio"], errors="coerce")
    if "volume" not in df.columns:
        return pd.Series(1.0, index=df.index)
    volume = pd.to_numeric(df["volume"], errors="coerce")
    if "volume_ma20" in df.columns:
        base = pd.to_numeric(df["volume_ma20"], errors="coerce")
    else:
        base = volume.groupby(df["fund_code"]).transform(lambda s: s.rolling(20, min_periods=3).mean())
    return (volume / base.replace(0, np.nan)).replace([np.inf, -np.inf], np.nan).fillna(1.0)


def _volume_ratio_window(df: pd.DataFrame, window: int) -> pd.Series:
    if "volume" not in df.columns:
        return pd.Series(1.0, index=df.index)
    volume = pd.to_numeric(df["volume"], errors="coerce")
    base = volume.groupby(df["fund_code"]).transform(lambda s: s.rolling(window, min_periods=3).mean())
    return (volume / base.replace(0, np.nan)).replace([np.inf, -np.inf], np.nan).fillna(1.0)


def _bid_ask_spread_pct(df: pd.DataFrame, value_col: str | None) -> pd.Series:
    if "bid_ask_spread_pct" in df.columns:
        return pd.to_numeric(df["bid_ask_spread_pct"], errors="coerce")
    if "bid_ask_spread" not in df.columns or value_col is None:
        return pd.Series(0.0, index=df.index)
    spread = pd.to_numeric(df["bid_ask_spread"], errors="coerce")
    price = pd.to_numeric(df[value_col], errors="coerce")
    return (spread / price.replace(0, np.nan) * 100.0).replace([np.inf, -np.inf], np.nan).fillna(0.0)


def _market_beta(fund_returns: pd.Series, df: pd.DataFrame) -> pd.Series:
    market = _coalesce(df, ["market_change_pct", "market_return_pct", "index_change_pct"])
    return (fund_returns / market.replace(0, np.nan)).clip(-3, 3).fillna(0.0)


def _flow_signal(df: pd.DataFrame) -> pd.Series:
    flow = _coalesce(df, ["fund_flow_pct", "etf_flow_pct", "net_inflow_pct"])
    premium = _coalesce(df, ["nav_premium_pct", "premium_pct"])
    change = _coalesce(df, ["change_pct", "daily_change_pct"])
    return (flow * 0.45 + premium * 0.25 + change * 0.30).fillna(0.0)


def _holding_exposure(df: pd.DataFrame) -> pd.Series:
    holding = _coalesce(df, ["holding_weighted_change_pct", "top_holding_change_pct"])
    market = _coalesce(df, ["market_change_pct", "index_change_pct"])
    sector = _coalesce(df, ["sector_change_pct", "industry_change_pct"])
    return (holding * 0.55 + market * 0.25 + sector * 0.20).fillna(0.0)


def _intraday_liquidity(df: pd.DataFrame) -> pd.Series:
    volume_ratio = _volume_ratio(df)
    turnover = _coalesce(df, ["turnover_rate", "turnover_pct"])
    spread = _coalesce(df, ["bid_ask_spread_pct", "spread_pct"])
    return (abs(volume_ratio - 1.0) + turnover * 0.15 - spread * 0.20).fillna(0.0)


def _etf_flow_proxy(df: pd.DataFrame) -> pd.Series:
    flow = _coalesce(df, ["etf_flow_pct", "fund_flow_pct", "net_inflow_pct"])
    premium = _coalesce(df, ["nav_premium_pct", "premium_pct"])
    volume_ratio = _volume_ratio(df)
    return (flow * 0.55 + premium * 0.25 + (volume_ratio - 1.0) * 0.20).fillna(0.0)
