from __future__ import annotations

from typing import Any

from .common import (
    current_available_time,
    datetime_series,
    enforce_contract,
    first_existing,
    infer_market_from_name,
    numeric_series,
    read_tracking_map,
    require_pandas,
    standardize_code,
    text_series,
)


def require_akshare():
    try:
        import akshare as ak
    except ImportError as exc:
        raise SystemExit(
            "Missing optional dependency: akshare. "
            "Use Python 3.11/3.12 and run `pip install -e .[data]` or `pip install akshare`."
        ) from exc
    return ak


def collect_etf_universe(tracking_map_path: str | None = None, skip_validation: bool = False):
    ak = require_akshare()
    pd = require_pandas()
    raw = _call_first_available(ak, ("fund_etf_spot_em", "fund_etf_spot_ths"))
    tracking_map = read_tracking_map(tracking_map_path)

    code_col = first_existing(raw.columns, ("代码", "基金代码", "symbol", "code"))
    name_col = first_existing(raw.columns, ("名称", "基金简称", "name"))
    if code_col is None or name_col is None:
        raise ValueError(f"Cannot infer ETF code/name columns from AkShare output: {list(raw.columns)}")

    rows: list[dict[str, Any]] = []
    for _, row in raw.iterrows():
        fund_code = standardize_code(row[code_col])
        fund_name = str(row[name_col]).strip()
        mapped = tracking_map.get(fund_code, {})
        rows.append({
            "fund_code": fund_code,
            "fund_name": fund_name,
            "fund_type": "ETF",
            "tracking_index": mapped.get("tracking_index") or "UNMAPPED",
            "market": mapped.get("market") or infer_market_from_name(fund_name),
            "is_etf": True,
            "is_lof": False,
            "fee_rate": None,
            "inception_date": None,
        })
    return enforce_contract(pd.DataFrame(rows), "dim_fund", skip_validation=skip_validation)


def collect_etf_spot(skip_validation: bool = False):
    ak = require_akshare()
    pd = require_pandas()
    raw = _call_first_available(ak, ("fund_etf_spot_em", "fund_etf_spot_ths"))
    code = text_series(raw, ("代码", "基金代码", "symbol", "code")).map(standardize_code)
    price = numeric_series(raw, ("最新价", "最新", "现价", "price"))
    bid = numeric_series(raw, ("买一", "买一价", "bid"))
    ask = numeric_series(raw, ("卖一", "卖一价", "ask"))
    available_time = current_available_time()

    out = pd.DataFrame({
        "fund_code": code,
        "timestamp": available_time,
        "available_time": available_time,
        "price": price,
        "iopv": numeric_series(raw, ("IOPV实时估值", "IOPV", "基金净值", "估值")),
        "premium_pct": numeric_series(raw, ("基金折价率", "溢价率", "折价率", "premium_pct")),
        "volume": numeric_series(raw, ("成交量", "volume")),
        "amount": numeric_series(raw, ("成交额", "amount")),
        "bid_ask_spread": ask - bid,
    })
    return enforce_contract(out, "fund_intraday", skip_validation=skip_validation)


def collect_etf_intraday(
    symbol: str,
    period: str = "1",
    start_date: str | None = None,
    end_date: str | None = None,
    adjust: str = "",
    skip_validation: bool = False,
):
    ak = require_akshare()
    pd = require_pandas()
    kwargs = {"symbol": symbol, "period": period, "adjust": adjust}
    if start_date:
        kwargs["start_date"] = start_date
    if end_date:
        kwargs["end_date"] = end_date
    raw = ak.fund_etf_hist_min_em(**kwargs)
    timestamp = datetime_series(raw, ("时间", "日期", "timestamp", "time"))

    out = pd.DataFrame({
        "fund_code": standardize_code(symbol),
        "timestamp": timestamp,
        "available_time": timestamp,
        "price": numeric_series(raw, ("收盘", "最新价", "close", "price")),
        "iopv": numeric_series(raw, ("IOPV实时估值", "IOPV", "iopv")),
        "premium_pct": numeric_series(raw, ("基金折价率", "溢价率", "折价率", "premium_pct")),
        "volume": numeric_series(raw, ("成交量", "volume")),
        "amount": numeric_series(raw, ("成交额", "amount")),
        "bid_ask_spread": numeric_series(raw, ("买卖价差", "bid_ask_spread")),
    })
    return enforce_contract(out, "fund_intraday", skip_validation=skip_validation)


def collect_etf_daily(symbol: str, start_date: str, end_date: str, adjust: str = "", skip_validation: bool = False):
    pd = require_pandas()
    raw = _call_etf_daily(symbol=symbol, start_date=start_date, end_date=end_date, adjust=adjust)
    close = numeric_series(raw, ("收盘", "close"))
    trade_date = datetime_series(raw, ("日期", "date", "trade_date"))
    out = pd.DataFrame({
        "fund_code": standardize_code(symbol),
        "trade_date": trade_date,
        "available_time": _market_close_time(trade_date),
        "nav": close,
        "adjusted_nav": close,
        "estimated_nav": None,
        "share": None,
        "aum": None,
        "flow": None,
    })
    return enforce_contract(out, "fund_daily", skip_validation=skip_validation)


def _call_etf_daily(symbol: str, start_date: str, end_date: str, adjust: str):
    ak = require_akshare()
    errors: list[str] = []
    if hasattr(ak, "fund_etf_hist_em"):
        try:
            return ak.fund_etf_hist_em(
                symbol=symbol,
                period="daily",
                start_date=start_date,
                end_date=end_date,
                adjust=adjust,
            )
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"fund_etf_hist_em: {exc}")
    if hasattr(ak, "fund_etf_hist_sina"):
        try:
            raw = ak.fund_etf_hist_sina(symbol=_sina_exchange_symbol(symbol))
            pd = require_pandas()
            date_col = first_existing(raw.columns, ("date", "日期"))
            if date_col is not None:
                dates = pd.to_datetime(raw[date_col], errors="coerce")
                start = pd.to_datetime(start_date, format="%Y%m%d", errors="coerce")
                end = pd.to_datetime(end_date, format="%Y%m%d", errors="coerce")
                if pd.notna(start):
                    raw = raw.loc[dates >= start]
                if pd.notna(end):
                    raw = raw.loc[dates <= end]
            return raw
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"fund_etf_hist_sina: {exc}")
    raise RuntimeError(f"AkShare ETF daily collection failed for {symbol}. {'; '.join(errors)}")


def _sina_exchange_symbol(symbol: str) -> str:
    code = standardize_code(symbol)
    if code.startswith(("5", "6")):
        return f"sh{code}"
    if code.startswith(("0", "1", "2", "3")):
        return f"sz{code}"
    return code


def collect_index_daily(symbol: str, start_date: str | None = None, end_date: str | None = None, skip_validation: bool = False):
    ak = require_akshare()
    pd = require_pandas()
    raw = _call_index_daily(ak, symbol=symbol, start_date=start_date, end_date=end_date)
    trade_date = datetime_series(raw, ("日期", "date", "trade_date"))
    out = pd.DataFrame({
        "index_code": symbol,
        "trade_date": trade_date,
        "available_time": _market_close_time(trade_date),
        "open": numeric_series(raw, ("开盘", "open")),
        "high": numeric_series(raw, ("最高", "high")),
        "low": numeric_series(raw, ("最低", "low")),
        "close": numeric_series(raw, ("收盘", "close")),
        "volume": numeric_series(raw, ("成交量", "volume")),
        "amount": numeric_series(raw, ("成交额", "amount")),
        "valuation": numeric_series(raw, ("估值", "valuation")),
    })
    return enforce_contract(out, "index_daily", skip_validation=skip_validation)


def collect_index_intraday(
    symbol: str,
    period: str = "1",
    start_date: str | None = None,
    end_date: str | None = None,
    skip_validation: bool = False,
):
    ak = require_akshare()
    pd = require_pandas()
    raw = _call_index_intraday(ak, symbol=symbol, period=period, start_date=start_date, end_date=end_date)
    timestamp = datetime_series(raw, ("时间", "日期", "timestamp", "time"))
    out = pd.DataFrame({
        "index_code": symbol,
        "timestamp": timestamp,
        "available_time": timestamp,
        "price": numeric_series(raw, ("收盘", "最新价", "close", "price")),
        "return": numeric_series(raw, ("涨跌幅", "return", "change_pct")),
        "volume": numeric_series(raw, ("成交量", "volume")),
        "amount": numeric_series(raw, ("成交额", "amount")),
    })
    return enforce_contract(out, "index_intraday", skip_validation=skip_validation)


def collect_futures_main(
    symbol: str,
    start_date: str,
    end_date: str,
    underlying: str | None = None,
    skip_validation: bool = False,
):
    ak = require_akshare()
    pd = require_pandas()
    raw = ak.futures_main_sina(symbol=symbol, start_date=start_date, end_date=end_date)
    timestamp = datetime_series(raw, ("日期", "date", "trade_date"))
    out = pd.DataFrame({
        "contract": symbol,
        "underlying": underlying or symbol,
        "timestamp": timestamp,
        "available_time": _market_close_time(timestamp),
        "price": numeric_series(raw, ("收盘价", "收盘", "close", "price")),
        "basis": numeric_series(raw, ("基差", "basis")),
        "open_interest": numeric_series(raw, ("持仓量", "持仓", "open_interest")),
        "term_structure": numeric_series(raw, ("期限结构", "term_structure")),
    })
    return enforce_contract(out, "futures_bar", skip_validation=skip_validation)


def _market_close_time(values, close_time: str = "16:00:00"):
    pd = require_pandas()
    parsed = pd.to_datetime(values, errors="coerce")
    return parsed.dt.strftime(f"%Y-%m-%d {close_time}")


def _call_first_available(ak, names: tuple[str, ...]):
    last_error: Exception | None = None
    for name in names:
        fn = getattr(ak, name, None)
        if fn is None:
            continue
        try:
            return fn()
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            last_error = exc
    available = ", ".join(names)
    raise RuntimeError(f"None of the AkShare functions succeeded: {available}. Last error: {last_error}")


def _call_index_daily(ak, symbol: str, start_date: str | None, end_date: str | None):
    errors: list[str] = []
    if hasattr(ak, "stock_zh_index_daily_em"):
        try:
            return ak.stock_zh_index_daily_em(symbol=symbol)
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"stock_zh_index_daily_em: {exc}")
    if hasattr(ak, "stock_zh_index_daily"):
        try:
            return ak.stock_zh_index_daily(symbol=symbol)
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"stock_zh_index_daily: {exc}")
    if hasattr(ak, "stock_zh_index_hist_csindex"):
        kwargs = {"symbol": symbol}
        if start_date:
            kwargs["start_date"] = start_date
        if end_date:
            kwargs["end_date"] = end_date
        try:
            return ak.stock_zh_index_hist_csindex(**kwargs)
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"stock_zh_index_hist_csindex: {exc}")
    detail = "; ".join(errors) if errors else "No known AkShare index daily function found."
    raise RuntimeError(f"AkShare index daily collection failed for {symbol}. {detail}")


def _call_index_intraday(ak, symbol: str, period: str, start_date: str | None, end_date: str | None):
    errors: list[str] = []
    if hasattr(ak, "index_zh_a_hist_min_em"):
        kwargs = {"symbol": symbol, "period": period}
        if start_date:
            kwargs["start_date"] = start_date
        if end_date:
            kwargs["end_date"] = end_date
        try:
            return ak.index_zh_a_hist_min_em(**kwargs)
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"index_zh_a_hist_min_em: {exc}")
    if hasattr(ak, "stock_zh_index_spot_em"):
        try:
            return ak.stock_zh_index_spot_em()
        except Exception as exc:  # pragma: no cover - depends on remote source behavior
            errors.append(f"stock_zh_index_spot_em: {exc}")
    detail = "; ".join(errors) if errors else "No known AkShare index intraday function found."
    raise RuntimeError(f"AkShare index intraday collection failed for {symbol}. {detail}")
