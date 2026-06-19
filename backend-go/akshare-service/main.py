"""AKShare 微服务 - 通过 HTTP API 提供市场行情数据。

本服务基于 FastAPI 构建，封装 AKShare 数据源，为后端 Go 服务提供
A 股指数行情、分钟线、K 线、北向资金流向及股票列表同步等接口。
所有业务接口均需通过 Bearer Token 认证。
"""
import hmac
import logging
import os
from typing import Annotated

from fastapi import Depends, FastAPI, Header, HTTPException, Query
import akshare as ak
import pandas as pd
import requests
from datetime import datetime

app = FastAPI(title="AKShare Data Service", version="1.0.0")
logger = logging.getLogger("akshare-service")
SERVICE_TOKEN = os.getenv("AKSHARE_SERVICE_TOKEN", "")

CN_INDEX_CODES = {"000001", "399001", "399006"}
TENCENT_CN_INDEX_SYMBOLS = {
    "000001": "sh000001",
    "399001": "sz399001",
    "399006": "sz399006",
}


def require_service_token(
    authorization: Annotated[str | None, Header()] = None,
):
    """验证请求携带的服务间认证 Token。

    通过比较 Authorization 请求头与预设的 AKSHARE_SERVICE_TOKEN 环境变量
    来完成 Bearer Token 认证，使用 hmac.compare_digest 防止时序攻击。

    Args:
        authorization: HTTP Authorization 请求头，预期格式为 "Bearer <token>"。

    Raises:
        HTTPException: 当服务未配置 Token 时返回 503；当 Token 不匹配时返回 401。
    """
    if not SERVICE_TOKEN:
        raise HTTPException(status_code=503, detail="service authentication is not configured")
    expected = f"Bearer {SERVICE_TOKEN}"
    if authorization is None or not hmac.compare_digest(authorization, expected):
        raise HTTPException(status_code=401, detail="unauthorized")


def safe_float(value):
    """安全地将值转换为浮点数。

    当值为 NaN 或空字符串时返回 0.0，避免 pandas 缺失值传播。

    Args:
        value: 待转换的值，通常来自 DataFrame 单元格。

    Returns:
        float: 转换后的浮点数，缺失时为 0.0。
    """
    return float(value) if pd.notna(value) and str(value).strip() != "" else 0.0


def format_cn_index_quotes(df):
    df = df[df["代码"].isin(CN_INDEX_CODES)]
    result = []
    for _, row in df.iterrows():
        result.append({
            "code": row["代码"],
            "name": row["名称"],
            "price": safe_float(row["最新价"]),
            "change_pct": safe_float(row["涨跌幅"]),
            "volume": safe_float(row["成交量"]),
        })
    return result


def parse_tencent_index_payload(payload):
    """解析腾讯财经指数行情的原始响应数据。

    腾讯接口返回 GBK 编码的 JavaScript 变量赋值语句，
    本函数将其解析为标准化的指数行情字典列表。

    Args:
        payload: 腾讯接口返回的原始字节流。

    Returns:
        list[dict]: 包含 code、name、price、change_pct、volume 字段的字典列表。
    """
    text = payload.decode("gbk", errors="ignore").strip("\ufeff \t\r\n")
    result = []
    for statement in text.split(";"):
        start = statement.find('"')
        end = statement.rfind('"')
        if start < 0 or end <= start:
            continue
        symbol_key = statement.split("=", 1)[0].strip().removeprefix("v_")
        fields = statement[start + 1:end].split("~")
        if len(fields) < 35:
            continue
        code = fields[2].strip()
        if TENCENT_CN_INDEX_SYMBOLS.get(code) != symbol_key:
            continue
        result.append({
            "code": code,
            "name": fields[1].strip(),
            "price": safe_float(fields[3]),
            "change_pct": safe_float(fields[32]),
            "volume": safe_float(fields[6]),
        })
    return result


def fetch_tencent_index_quotes():
    """通过腾讯财经接口获取 A 股指数行情（AKShare 备用数据源）。

    当 AKShare 主数据源不可用或返回空数据时，回退至腾讯财经接口
    获取上证指数、深证成指、创业板指的实时行情。

    Returns:
        list[dict]: 包含 code、name、price、change_pct、volume 字段的字典列表。

    Raises:
        RuntimeError: 当腾讯接口返回空数据时抛出。
    """
    symbols = ",".join(TENCENT_CN_INDEX_SYMBOLS.values())
    response = requests.get(
        f"https://qt.gtimg.cn/q={symbols}",
        headers={"Referer": "https://gu.qq.com/"},
        timeout=10,
    )
    response.raise_for_status()
    result = parse_tencent_index_payload(response.content)
    if not result:
        raise RuntimeError("empty Tencent index quote fallback")
    return result

@app.get("/health")
async def health():
    """健康检查接口。

    Returns:
        dict: 包含 status 和 timestamp 字段的健康状态信息。
    """
    return {"status": "ok", "timestamp": datetime.now().isoformat()}

@app.get("/api/v1/index/quote", dependencies=[Depends(require_service_token)])
async def index_quote(market: str = "cn"):
    """获取指数实时行情。

    优先使用 AKShare 数据源，当其不可用或返回空数据时
    自动回退至腾讯财经接口。当前仅支持 A 股市场。

    Args:
        market: 市场标识，默认 "cn"（A 股）。

    Returns:
        dict: code=0 时 data 为行情列表；code=-1 时为不支持的市场提示。

    Raises:
        HTTPException: 当所有数据源均不可用时返回 502。
    """
    try:
        if market == "cn":
            try:
                result = format_cn_index_quotes(ak.stock_zh_index_spot_em())
                if not result:
                    raise RuntimeError("empty AKShare index quote result")
            except Exception:
                result = fetch_tencent_index_quotes()
            return {"code": 0, "data": result}
        return {"code": -1, "message": f"market {market} not supported"}
    except Exception:
        logger.exception("index quote request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")

@app.get("/api/v1/index/minute", dependencies=[Depends(require_service_token)])
async def index_minute(code: str, market: str = "cn"):
    """获取指数分钟线数据。

    返回指定指数的 1 分钟级别分时行情。

    Args:
        code: 指数代码，如 "000001"（上证指数）。
        market: 市场标识，默认 "cn"（A 股）。

    Returns:
        dict: code=0 时 data 为分钟线列表（含 time、price、volume）；
              code=-1 时为不支持的市场提示。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        if market == "cn":
            df = ak.stock_zh_index_minute_em(symbol=code, period="1")
            result = []
            for _, row in df.iterrows():
                result.append({
                    "time": row["day"],
                    "price": float(row["price"]) if pd.notna(row["price"]) else 0,
                    "volume": float(row["volume"]) if pd.notna(row["volume"]) else 0,
                })
            return {"code": 0, "data": result}
        return {"code": -1, "message": f"market {market} not supported"}
    except Exception:
        logger.exception("index minute request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")

@app.get("/api/v1/index/kline", dependencies=[Depends(require_service_token)])
async def index_kline(
    code: str,
    market: str = "cn",
    count: int = Query(default=120, ge=1, le=500),
):
    """获取指数日 K 线数据。

    返回指定指数最近 N 个交易日的开高低收及成交量数据。

    Args:
        code: 指数代码，如 "000001"（上证指数）。
        market: 市场标识，默认 "cn"（A 股）。
        count: 返回的 K 线条数，默认 120，范围 1-500。

    Returns:
        dict: code=0 时 data 为 K 线列表（含 date、open、close、high、low、volume）；
              code=-1 时为不支持的市场提示。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        if market == "cn":
            df = ak.stock_zh_index_daily_em(symbol=code)
            df = df.tail(count)
            result = []
            for _, row in df.iterrows():
                result.append({
                    "date": str(row["日期"]),
                    "open": float(row["开盘"]) if pd.notna(row["开盘"]) else 0,
                    "close": float(row["收盘"]) if pd.notna(row["收盘"]) else 0,
                    "high": float(row["最高"]) if pd.notna(row["最高"]) else 0,
                    "low": float(row["最低"]) if pd.notna(row["最低"]) else 0,
                    "volume": float(row["成交量"]) if pd.notna(row["成交量"]) else 0,
                })
            return {"code": 0, "data": result}
        return {"code": -1, "message": f"market {market} not supported"}
    except Exception:
        logger.exception("index kline request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")

@app.get("/api/v1/northbound/flow", dependencies=[Depends(require_service_token)])
async def northbound_flow():
    """获取北向资金流向数据。

    返回当日沪股通、深股通净买入额及分钟级分时流向时间线。
    分钟线数据来自 AKShare 实时接口，日汇总数据来自历史接口。
    当所有数据均为空时返回零值占位结果。

    Returns:
        dict: code=0 时 data 包含 sh_net_buy、sz_net_buy、total_net_buy 和 timeline。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        empty_result = {"sh_net_buy": 0, "sz_net_buy": 0, "total_net_buy": 0, "timeline": []}

        # Minute-level intraday timeline
        timeline = []
        try:
            df_min = ak.stock_hsgt_fund_min_em(symbol="北向资金")
            # Drop rows where both sh and sz are NaN
            df_min = df_min.dropna(subset=["沪股通", "深股通"], how="all")
            now = datetime.now()
            current_time = now.strftime("%H:%M")
            for _, row in df_min.iterrows():
                t = str(row["时间"]).strip()
                if len(t) >= 5 and ":" in t:
                    t = t[:5]
                # Skip future time points beyond current time
                if t > current_time:
                    continue
                sh = safe_float(row.get("沪股通", 0))
                sz = safe_float(row.get("深股通", 0))
                # Allow zero values for intraday timeline (preserve all time points)
                timeline.append({
                    "time": t,
                    "sh_flow": sh,
                    "sz_flow": sz,
                })
        except Exception as e:
            logger.debug(f"failed to fetch northbound minute data: {e}")
            pass

        # Daily summary totals
        sh_net_buy = 0.0
        sz_net_buy = 0.0
        total_net_buy = 0.0
        try:
            df_hist = ak.stock_hsgt_hist_em(symbol="北向资金")
            if not df_hist.empty:
                last = df_hist.iloc[-1]
                total_net_buy = safe_float(last.get("当日资金流入", 0))
                # Try to get sh/sz breakdown from available columns
                sh_net_buy = safe_float(last.get("沪股通净买额", last.get("沪股通", 0)))
                sz_net_buy = safe_float(last.get("深股通净买额", last.get("深股通", 0)))
        except Exception:
            pass

        if not timeline and sh_net_buy == 0.0 and sz_net_buy == 0.0 and total_net_buy == 0.0:
            return {"code": 0, "data": empty_result}

        return {
            "code": 0,
            "data": {
                "sh_net_buy": sh_net_buy,
                "sz_net_buy": sz_net_buy,
                "total_net_buy": total_net_buy,
                "timeline": timeline,
            },
        }
    except Exception:
        logger.exception("northbound flow request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")


@app.get("/api/v1/hsgt/hist", dependencies=[Depends(require_service_token)])
async def hsgt_hist(
    symbol: str = Query(default="北向资金", description="choice of {北向资金, 沪股通, 深股通, 南向资金, 港股通沪, 港股通深}"),
    days: int = Query(default=365, ge=1, le=730, description="返回最近 N 个交易日的数据"),
):
    """获取沪深港通历史资金流向数据。

    通过 AKShare 的 stock_hsgt_hist_em 接口获取东方财富沪深港通历史数据，
    支持北向资金（沪股通+深股通）和南向资金（港股通沪+港股通深）。

    Args:
        symbol: 数据类型，可选 "北向资金"、"沪股通"、"深股通"、"南向资金"、"港股通沪"、"港股通深"。
        days: 返回最近 N 个交易日的数据，默认 365，范围 1-730。

    Returns:
        dict: code=0 时 data 为历史数据列表，每条包含 date、net_buy、buy_amount、sell_amount 等字段。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        df = ak.stock_hsgt_hist_em(symbol=symbol)
        if df.empty:
            return {"code": 0, "data": []}
        df = df.tail(days)
        result = []
        for _, row in df.iterrows():
            result.append({
                "date": str(row["日期"]),
                "net_buy": safe_float(row.get("当日成交净买额", 0)),
                "buy_amount": safe_float(row.get("买入成交额", 0)),
                "sell_amount": safe_float(row.get("卖出成交额", 0)),
                "acc_net_buy": safe_float(row.get("历史累计净买额", 0)),
                "cash_in": safe_float(row.get("当日资金流入", 0)),
                "balance": safe_float(row.get("当日余额", 0)),
            })
        return {"code": 0, "data": result}
    except Exception:
        logger.exception("hsgt hist request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")


@app.post("/api/v1/stock/sync", dependencies=[Depends(require_service_token)])
async def stock_sync():
    """同步 A 股股票列表。

    从 AKShare 获取全部 A 股股票的代码、名称和所属行业，
    用于后端数据库的全量同步。

    Returns:
        dict: code=0 时 data 为股票列表（含 stock_code、stock_name、industry）。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        df = ak.stock_zh_a_spot_em()
        result = []
        for _, row in df.iterrows():
            result.append({
                "stock_code": row["代码"],
                "stock_name": row["名称"],
                "industry": row.get("行业", ""),
            })
        return {"code": 0, "data": result}
    except Exception:
        logger.exception("stock sync request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")

@app.get("/api/v1/stock/restricted", dependencies=[Depends(require_service_token)])
async def stock_restricted(code: str):
    """获取个股限售解禁数据。

    调用 AKShare 的 stock_restricted_release_queue_sina 接口，
    返回指定股票的历史及未来解禁记录。

    Args:
        code: 6 位股票代码，如 "000001"。

    Returns:
        dict: code=0 时 data 为解禁记录列表，每条包含 date、volume、market_value、batch、announce_date。

    Raises:
        HTTPException: 当数据源不可用时返回 502。
    """
    try:
        # 构造 sina 格式 symbol：6 开头为 sh，否则为 sz
        prefix = "sh" if code.startswith("6") else "sz"
        symbol = f"{prefix}{code}"
        df = ak.stock_restricted_release_queue_sina(symbol=symbol)
        if df.empty:
            return {"code": 0, "data": []}
        result = []
        for _, row in df.iterrows():
            # 日期转换为 YYYY-MM-DD 字符串
            release_date = row.get("解禁日期")
            if hasattr(release_date, "strftime"):
                release_date = release_date.strftime("%Y-%m-%d")
            else:
                release_date = str(release_date) if pd.notna(release_date) else ""

            announce_date = row.get("公告日期")
            if hasattr(announce_date, "strftime"):
                announce_date = announce_date.strftime("%Y-%m-%d")
            else:
                announce_date = str(announce_date) if pd.notna(announce_date) else ""

            result.append({
                "date": release_date,
                "volume": safe_float(row.get("解禁数量", 0)),
                "market_value": safe_float(row.get("解禁股流通市值", 0)),
                "batch": int(row.get("上市批次", 0)) if pd.notna(row.get("上市批次")) else 0,
                "announce_date": announce_date,
            })
        return {"code": 0, "data": result}
    except Exception:
        logger.exception("stock restricted request failed")
        raise HTTPException(status_code=502, detail="upstream market data unavailable")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        app,
        host=os.getenv("AKSHARE_HOST", "127.0.0.1"),
        port=int(os.getenv("AKSHARE_PORT", "8900")),
    )
