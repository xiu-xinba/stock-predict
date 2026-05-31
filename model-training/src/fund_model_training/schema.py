from __future__ import annotations

from typing import Final

LABEL_TO_ID: Final[dict[str, int]] = {
    "down": 0,
    "flat": 1,
    "up": 2,
}

ID_TO_LABEL: Final[dict[int, str]] = {v: k for k, v in LABEL_TO_ID.items()}

BACKEND_V1_FEATURES: Final[list[str]] = [
    "momentum_5d",
    "return_10d",
    "volatility_20d",
    "volume_ratio",
    "market_beta",
    "sector_momentum",
    "flow_signal",
    "mean_reversion",
]

EXTENDED_V1_FEATURES: Final[list[str]] = [
    *BACKEND_V1_FEATURES,
    "holding_exposure",
    "intraday_liquidity",
    "etf_flow_proxy",
]

INDEX_FUND_DAILY_V1_FEATURES: Final[list[str]] = [
    *EXTENDED_V1_FEATURES,
    "fund_return_1d",
    "fund_return_5d",
    "fund_volatility_20d",
    "index_return_1d",
    "index_return_5d",
    "index_volatility_20d",
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
]

INDEX_FUND_INTRADAY_V1_FEATURES: Final[list[str]] = [
    "fund_return_1m",
    "fund_return_3m",
    "fund_return_5m",
    "fund_volatility_15m",
    "fund_volume_ratio_20m",
    "premium_pct",
    "bid_ask_spread_pct",
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
]

ALL_FUND_NAV_DAILY_V1_FEATURES: Final[list[str]] = [
    "fund_return_1d",
    "fund_return_5d",
    "fund_return_20d",
    "fund_volatility_20d",
    "fund_drawdown_20d",
    "mean_reversion",
]

FEATURE_SETS: Final[dict[str, list[str]]] = {
    "backend_v1": BACKEND_V1_FEATURES,
    "extended_v1": EXTENDED_V1_FEATURES,
    "index_fund_daily_v1": INDEX_FUND_DAILY_V1_FEATURES,
    "index_fund_intraday_v1": INDEX_FUND_INTRADAY_V1_FEATURES,
    "all_fund_nav_daily_v1": ALL_FUND_NAV_DAILY_V1_FEATURES,
}

REQUIRED_COLUMNS: Final[list[str]] = ["fund_code", "asof_time"]


def get_feature_names(feature_set: str) -> list[str]:
    try:
        return FEATURE_SETS[feature_set]
    except KeyError as exc:
        known = ", ".join(sorted(FEATURE_SETS))
        raise ValueError(f"Unknown feature_set '{feature_set}'. Known values: {known}") from exc
