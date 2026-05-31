from __future__ import annotations

import os
import time
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

MOOTDX_PAGE_SIZE = 800
MOOTDX_DEFAULT_PAGES = 4


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
    pd = require_pandas()
    raw = None
    if period in {"1", "5", "1m", "5m"}:
        try:
            raw = _call_mootdx_std_bars(
                symbol=standardize_code(symbol),
                period=period,
                start_date=start_date,
                end_date=end_date,
            )
        except Exception:
            raw = None
    if raw is None or raw.empty:
        ak = require_akshare()
        kwargs = {"symbol": symbol, "period": period, "adjust": adjust}
        if start_date:
            kwargs["start_date"] = start_date
        if end_date:
            kwargs["end_date"] = end_date
        try:
            raw = ak.fund_etf_hist_min_em(**kwargs)
        except Exception:
            if period != "1":
                raise
            code = standardize_code(symbol)
            try:
                raw = _call_eastmoney_trends(
                    symbol=code,
                    market_ids=(_market_id_for_cn_symbol(code),),
                    start_date=start_date,
                    end_date=end_date,
                )
            except Exception:
                raw = _call_tencent_minute(
                    quote_symbol=_tencent_fund_symbol(code),
                    start_date=start_date,
                    end_date=end_date,
                )
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


def _market_id_for_cn_symbol(symbol: str) -> int:
    code = standardize_code(symbol)
    return 1 if code.startswith(("5", "6")) else 0


def _normalize_index_symbol(symbol: str) -> str:
    raw = str(symbol).strip()
    lowered = raw.lower()
    if lowered.startswith(("sh", "sz")):
        return raw[2:]
    return raw


def _candidate_index_market_ids(symbol: str) -> tuple[int, ...]:
    lowered = str(symbol).strip().lower()
    if lowered.startswith("sh"):
        return (1, 0, 47)
    if lowered.startswith("sz"):
        return (0, 1, 47)
    return (1, 0, 47)


def _tencent_fund_symbol(symbol: str) -> str:
    code = standardize_code(symbol)
    return f"sh{code}" if code.startswith(("5", "6")) else f"sz{code}"


def _tencent_index_symbol(symbol: str) -> str:
    raw = str(symbol).strip()
    lowered = raw.lower()
    if lowered.startswith(("sh", "sz", "hk")):
        return raw
    if raw.upper() in {"HSI", "HSTECH"}:
        return f"hk{raw.upper()}"
    code = _normalize_index_symbol(raw)
    return f"sz{code}" if code.startswith(("399", "3")) else f"sh{code}"


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
    pd = require_pandas()
    raw = None
    if period in {"1", "5", "1m", "5m"}:
        try:
            raw = _call_mootdx_index_bars(
                symbol=symbol,
                period=period,
                start_date=start_date,
                end_date=end_date,
            )
        except Exception:
            raw = None
    if raw is None or raw.empty:
        ak = require_akshare()
        try:
            raw = _call_index_intraday(ak, symbol=symbol, period=period, start_date=start_date, end_date=end_date)
        except Exception:
            if period != "1":
                raise
            try:
                raw = _call_eastmoney_trends(
                    symbol=_normalize_index_symbol(symbol),
                    market_ids=_candidate_index_market_ids(symbol),
                    start_date=start_date,
                    end_date=end_date,
                )
            except Exception:
                raw = _call_tencent_minute(
                    quote_symbol=_tencent_index_symbol(symbol),
                    start_date=start_date,
                    end_date=end_date,
                )
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


def _call_eastmoney_trends(
    symbol: str,
    market_ids: tuple[int, ...],
    start_date: str | None = None,
    end_date: str | None = None,
    retries: int = 3,
):
    last_error: Exception | None = None
    for market_id in market_ids:
        params = {
            "fields1": "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
            "fields2": "f51,f52,f53,f54,f55,f56,f57,f58",
            "ut": "7eea3edcaed734bea9cbfc24409ed989",
            "ndays": "5",
            "iscr": "0",
            "secid": f"{market_id}.{symbol}",
        }
        for attempt in range(retries):
            try:
                data_json = _eastmoney_get_json(params, trust_env=attempt % 2 == 0)
                frame = _eastmoney_trends_to_frame(data_json, start_date=start_date, end_date=end_date)
                if not frame.empty:
                    return frame
            except Exception as exc:  # pragma: no cover - network dependent
                last_error = exc
                time.sleep(0.6 * (attempt + 1))
    raise RuntimeError(f"Eastmoney trends fallback failed for {symbol}: {last_error}")


def _call_tencent_minute(quote_symbol: str, start_date: str | None = None, end_date: str | None = None):
    last_error: Exception | None = None
    for base_url in (
        "https://web.ifzq.gtimg.cn/appstock/app/minute/query",
        "http://web.ifzq.gtimg.cn/appstock/app/minute/query",
    ):
        try:
            data_json = _tencent_get_json(base_url, quote_symbol)
            frame = _tencent_minute_to_frame(data_json, quote_symbol, start_date=start_date, end_date=end_date)
            if not frame.empty:
                return frame
        except Exception as exc:  # pragma: no cover - network dependent
            last_error = exc
            time.sleep(0.5)
    raise RuntimeError(f"Tencent minute fallback failed for {quote_symbol}: {last_error}")


def _call_mootdx_std_bars(
    symbol: str,
    period: str = "1",
    start_date: str | None = None,
    end_date: str | None = None,
):
    try:
        from mootdx.quotes import Quotes
    except ImportError as exc:
        raise RuntimeError("mootdx is not installed") from exc

    client = Quotes.factory(market="std")
    return _mootdx_collect_pages(
        fetch_page=lambda start, offset: client.bars(
            symbol=standardize_code(symbol),
            frequency=_mootdx_frequency(period),
            start=start,
            offset=offset,
        ),
        start_date=start_date,
        end_date=end_date,
    )


def _call_mootdx_index_bars(
    symbol: str,
    period: str = "1",
    start_date: str | None = None,
    end_date: str | None = None,
):
    mapping = _mootdx_index_mapping(symbol)
    if mapping is None:
        raise RuntimeError(f"mootdx index mapping not configured for {symbol}")
    try:
        from mootdx.quotes import Quotes
    except ImportError as exc:
        raise RuntimeError("mootdx is not installed") from exc

    market, code = mapping
    client = Quotes.factory(market="ext")
    return _mootdx_collect_pages(
        fetch_page=lambda start, offset: client.bars(
            market=market,
            symbol=code,
            frequency=_mootdx_frequency(period),
            start=start,
            offset=offset,
        ),
        start_date=start_date,
        end_date=end_date,
    )


def _mootdx_collect_pages(fetch_page, start_date: str | None = None, end_date: str | None = None):
    pd = require_pandas()
    pages = _mootdx_page_count()
    frames = []
    cutoff = pd.to_datetime(start_date, errors="coerce") if start_date else None
    for page in range(pages):
        raw = fetch_page(page * MOOTDX_PAGE_SIZE, MOOTDX_PAGE_SIZE)
        if raw is None or raw.empty:
            break
        frame = _mootdx_bars_to_frame(raw)
        if frame.empty:
            break
        frames.append(frame)
        if cutoff is not None:
            earliest = pd.to_datetime(frame["时间"], errors="coerce").min()
            if pd.notna(earliest) and earliest <= cutoff:
                break
    if not frames:
        return pd.DataFrame(columns=["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"])
    return _filter_intraday_frame(
        pd.concat(frames, ignore_index=True),
        start_date=start_date,
        end_date=end_date,
    )


def _mootdx_bars_to_frame(raw):
    pd = require_pandas()
    if raw is None or raw.empty:
        return pd.DataFrame(columns=["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"])
    frame = raw.copy()
    time_values = frame["datetime"] if "datetime" in frame.columns else frame.index
    out = pd.DataFrame({
        "时间": pd.to_datetime(time_values, errors="coerce"),
        "开盘": pd.to_numeric(frame.get("open"), errors="coerce"),
        "收盘": pd.to_numeric(frame.get("close"), errors="coerce"),
        "最高": pd.to_numeric(frame.get("high"), errors="coerce"),
        "最低": pd.to_numeric(frame.get("low"), errors="coerce"),
        "成交量": pd.to_numeric(_first_frame_column(frame, ("volume", "vol", "position", "trade")), errors="coerce"),
        "成交额": pd.to_numeric(_first_frame_column(frame, ("amount",)), errors="coerce"),
    })
    out["均价"] = (out["开盘"] + out["收盘"]) / 2
    out = out.dropna(subset=["时间", "收盘"]).sort_values("时间")
    out = out.drop_duplicates(subset=["时间"], keep="last")
    out["时间"] = out["时间"].astype(str)
    return out.reset_index(drop=True)


def _filter_intraday_frame(frame, start_date: str | None = None, end_date: str | None = None):
    pd = require_pandas()
    out = frame.copy()
    out["时间"] = pd.to_datetime(out["时间"], errors="coerce")
    out = out.dropna(subset=["时间"])
    if start_date:
        start = pd.to_datetime(start_date, errors="coerce")
        if pd.notna(start):
            out = out.loc[out["时间"] >= start]
    if end_date:
        end = pd.to_datetime(end_date, errors="coerce")
        if pd.notna(end):
            out = out.loc[out["时间"] <= end]
    out = out.sort_values("时间").drop_duplicates(subset=["时间"], keep="last")
    out["时间"] = out["时间"].astype(str)
    return out.reset_index(drop=True)


def _first_frame_column(frame, names: tuple[str, ...]):
    pd = require_pandas()
    for name in names:
        if name in frame.columns:
            return frame[name]
    return pd.Series([0.0] * len(frame), index=frame.index)


def _mootdx_frequency(period: str) -> str:
    raw = str(period).lower().strip()
    if raw in {"1", "1m", "1min"}:
        return "1m"
    if raw in {"5", "5m", "5min"}:
        return "5m"
    raise ValueError(f"Unsupported mootdx intraday period: {period}")


def _mootdx_page_count() -> int:
    raw = os.environ.get("FUND_MOOTDX_INTRADAY_PAGES")
    if not raw:
        return MOOTDX_DEFAULT_PAGES
    try:
        return max(int(raw), 1)
    except ValueError:
        return MOOTDX_DEFAULT_PAGES


def _mootdx_index_mapping(symbol: str) -> tuple[int, str] | None:
    code = _normalize_index_symbol(symbol)
    csi_market = 62
    if code in {"000300", "000905", "000852", "000903", "000906"}:
        return csi_market, code
    return None


def _tencent_get_json(base_url: str, quote_symbol: str) -> dict[str, Any]:
    try:
        import requests
    except ImportError as exc:
        raise SystemExit("Missing dependency: requests. Run `pip install requests`.") from exc

    session = requests.Session()
    session.trust_env = False
    response = session.get(
        base_url,
        params={"code": quote_symbol},
        timeout=15,
        headers={"User-Agent": "Mozilla/5.0", "Accept": "application/json,text/plain,*/*"},
    )
    response.raise_for_status()
    return response.json()


def _tencent_minute_to_frame(
    data_json: dict[str, Any],
    quote_symbol: str,
    start_date: str | None = None,
    end_date: str | None = None,
):
    pd = require_pandas()
    columns = ["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"]
    raw_data = ((data_json.get("data") or {}).get(quote_symbol) or {}).get("data") or {}
    trade_date = str(raw_data.get("date") or "")
    minute_rows = raw_data.get("data") or []
    if not trade_date or not minute_rows:
        return pd.DataFrame(columns=columns)

    date_value = pd.to_datetime(trade_date, format="%Y%m%d", errors="coerce")
    if pd.isna(date_value):
        return pd.DataFrame(columns=columns)

    rows: list[dict[str, Any]] = []
    for item in minute_rows:
        parts = str(item).split()
        if len(parts) < 4:
            continue
        hhmm, price, volume, amount = parts[:4]
        timestamp = pd.to_datetime(f"{date_value.date()} {hhmm[:2]}:{hhmm[2:]}:00", errors="coerce")
        rows.append({
            "时间": timestamp,
            "开盘": price,
            "收盘": price,
            "最高": price,
            "最低": price,
            "成交量": volume,
            "成交额": amount,
            "均价": price,
        })
    frame = pd.DataFrame(rows, columns=columns)
    frame = frame.dropna(subset=["时间"]).sort_values("时间")
    if start_date:
        start = pd.to_datetime(start_date, errors="coerce")
        if pd.notna(start):
            frame = frame.loc[frame["时间"] >= start]
    if end_date:
        end = pd.to_datetime(end_date, errors="coerce")
        if pd.notna(end):
            frame = frame.loc[frame["时间"] <= end]
    for column in columns[1:]:
        frame[column] = pd.to_numeric(frame[column], errors="coerce")
    frame["时间"] = frame["时间"].astype(str)
    return frame.reset_index(drop=True)


def _eastmoney_get_json(params: dict[str, str], trust_env: bool = False) -> dict[str, Any]:
    try:
        import requests
    except ImportError as exc:
        raise SystemExit("Missing dependency: requests. Run `pip install requests`.") from exc

    session = requests.Session()
    session.trust_env = trust_env
    response = session.get(
        "https://push2his.eastmoney.com/api/qt/stock/trends2/get",
        params=params,
        timeout=15,
        headers={
            "User-Agent": "Mozilla/5.0",
            "Referer": "https://quote.eastmoney.com/",
            "Accept": "application/json,text/plain,*/*",
        },
    )
    response.raise_for_status()
    return response.json()


def _eastmoney_trends_to_frame(data_json: dict[str, Any], start_date: str | None = None, end_date: str | None = None):
    pd = require_pandas()
    trends = (data_json.get("data") or {}).get("trends") or []
    columns = ["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"]
    if not trends:
        return pd.DataFrame(columns=columns)
    temp_df = pd.DataFrame([item.split(",") for item in trends]).iloc[:, :8]
    temp_df.columns = columns
    temp_df["时间"] = pd.to_datetime(temp_df["时间"], errors="coerce")
    temp_df = temp_df.dropna(subset=["时间"]).sort_values("时间")
    if start_date:
        start = pd.to_datetime(start_date, errors="coerce")
        if pd.notna(start):
            temp_df = temp_df.loc[temp_df["时间"] >= start]
    if end_date:
        end = pd.to_datetime(end_date, errors="coerce")
        if pd.notna(end):
            temp_df = temp_df.loc[temp_df["时间"] <= end]
    for column in columns[1:]:
        temp_df[column] = pd.to_numeric(temp_df[column], errors="coerce")
    temp_df["时间"] = temp_df["时间"].astype(str)
    return temp_df.reset_index(drop=True)


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


def collect_open_fund_nav(
    symbol: str,
    start_date: str,
    end_date: str,
    skip_validation: bool = False,
):
    pd = require_pandas()
    ak = require_akshare()
    raw = None
    errors: list[str] = []
    if hasattr(ak, "fund_open_fund_info_em"):
        try:
            raw = ak.fund_open_fund_info_em(symbol=symbol, indicator="单位净值走势")
        except Exception as exc:
            errors.append(f"fund_open_fund_info_em: {exc}")
    if raw is None or raw.empty:
        raise RuntimeError(f"Open fund NAV collection failed for {symbol}. {'; '.join(errors)}")
    trade_date = datetime_series(raw, ("净值日期", "日期", "date", "trade_date"))
    nav = numeric_series(raw, ("单位净值", "净值", "nav", "value"))
    accumulated_nav = numeric_series(raw, ("累计净值", "accumulated_nav", "acc_nav"))
    out = pd.DataFrame({
        "fund_code": standardize_code(symbol),
        "trade_date": trade_date,
        "available_time": _market_close_time(trade_date),
        "nav": nav,
        "accumulated_nav": accumulated_nav if accumulated_nav.notna().any() else nav,
        "adjusted_nav": nav,
        "estimated_nav": None,
        "share": None,
        "aum": None,
        "flow": None,
    })
    if start_date:
        start = pd.to_datetime(start_date, errors="coerce")
        mask = pd.to_datetime(out["trade_date"], errors="coerce") >= start
        out = out.loc[mask]
    if end_date:
        end = pd.to_datetime(end_date, errors="coerce")
        mask = pd.to_datetime(out["trade_date"], errors="coerce") <= end
        out = out.loc[mask]
    return enforce_contract(out, "fund_daily", skip_validation=skip_validation)


def collect_money_fund_yield(
    symbol: str,
    start_date: str,
    end_date: str,
    skip_validation: bool = False,
):
    pd = require_pandas()
    ak = require_akshare()
    raw = None
    errors: list[str] = []
    if hasattr(ak, "fund_money_fund_daily_em"):
        try:
            raw = ak.fund_money_fund_daily_em(symbol=symbol, start_date=start_date, end_date=end_date)
        except Exception as exc:
            errors.append(f"fund_money_fund_daily_em: {exc}")
    if (raw is None or raw.empty) and hasattr(ak, "fund_open_fund_info_em"):
        try:
            raw = ak.fund_open_fund_info_em(symbol=symbol, indicator="单位净值走势")
        except Exception as exc:
            errors.append(f"fund_open_fund_info_em fallback: {exc}")
    if raw is None or raw.empty:
        raise RuntimeError(f"Money fund yield collection failed for {symbol}. {'; '.join(errors)}")
    trade_date = datetime_series(raw, ("净值日期", "日期", "date", "trade_date"))
    nav = numeric_series(raw, ("万份收益", "单位净值", "净值", "nav", "value"))
    annualized = numeric_series(raw, ("七日年化收益率", "七日年化", "annualized_yield"))
    out = pd.DataFrame({
        "fund_code": standardize_code(symbol),
        "trade_date": trade_date,
        "available_time": _market_close_time(trade_date),
        "nav": nav,
        "accumulated_nav": nav,
        "adjusted_nav": nav,
        "estimated_nav": None,
        "share": None,
        "aum": None,
        "flow": None,
        "annualized_yield_7d": annualized,
    })
    return enforce_contract(out, "fund_daily", skip_validation=skip_validation)


def collect_lof_daily(symbol: str, start_date: str, end_date: str, adjust: str = "", skip_validation: bool = False):
    pd = require_pandas()
    ak = require_akshare()
    raw = None
    errors: list[str] = []
    if hasattr(ak, "fund_lof_hist_em"):
        try:
            raw = ak.fund_lof_hist_em(symbol=symbol, period="daily", start_date=start_date, end_date=end_date, adjust=adjust)
        except Exception as exc:
            errors.append(f"fund_lof_hist_em: {exc}")
    if (raw is None or raw.empty) and hasattr(ak, "fund_etf_hist_em"):
        try:
            raw = ak.fund_etf_hist_em(symbol=symbol, period="daily", start_date=start_date, end_date=end_date, adjust=adjust)
        except Exception as exc:
            errors.append(f"fund_etf_hist_em fallback: {exc}")
    if raw is None or raw.empty:
        raise RuntimeError(f"LOF daily collection failed for {symbol}. {'; '.join(errors)}")
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
