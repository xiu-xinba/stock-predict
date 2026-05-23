from __future__ import annotations

from dataclasses import asdict, dataclass
from typing import Any, Iterable


@dataclass(frozen=True)
class ColumnSpec:
    name: str
    dtype: str
    required: bool
    description: str


@dataclass(frozen=True)
class TableSpec:
    name: str
    grain: str
    time_columns: tuple[str, ...]
    primary_keys: tuple[str, ...]
    columns: tuple[ColumnSpec, ...]

    @property
    def required_columns(self) -> tuple[str, ...]:
        return tuple(column.name for column in self.columns if column.required)


def c(name: str, dtype: str, required: bool, description: str) -> ColumnSpec:
    return ColumnSpec(name=name, dtype=dtype, required=required, description=description)


TABLE_SPECS: dict[str, TableSpec] = {
    "dim_fund": TableSpec(
        name="dim_fund",
        grain="fund",
        time_columns=("inception_date",),
        primary_keys=("fund_code",),
        columns=(
            c("fund_code", "string", True, "基金代码"),
            c("fund_name", "string", True, "基金名称"),
            c("fund_type", "string", True, "ETF/LOF/场外指数基金/联接基金/QDII"),
            c("tracking_index", "string", True, "跟踪指数代码"),
            c("market", "string", True, "CN/HK/US/GLOBAL"),
            c("is_etf", "bool", True, "是否为 ETF"),
            c("is_lof", "bool", False, "是否为 LOF"),
            c("fee_rate", "float", False, "综合费率"),
            c("inception_date", "datetime", False, "成立日期"),
        ),
    ),
    "fund_daily": TableSpec(
        name="fund_daily",
        grain="fund_day",
        time_columns=("trade_date", "available_time"),
        primary_keys=("fund_code", "trade_date"),
        columns=(
            c("fund_code", "string", True, "基金代码"),
            c("trade_date", "datetime", True, "交易日"),
            c("available_time", "datetime", True, "该记录在预测系统中可见的时间"),
            c("nav", "float", True, "单位净值"),
            c("adjusted_nav", "float", True, "复权净值"),
            c("estimated_nav", "float", False, "盘中估算净值"),
            c("share", "float", False, "基金份额"),
            c("aum", "float", False, "资产规模"),
            c("flow", "float", False, "申赎或资金流估计"),
        ),
    ),
    "fund_intraday": TableSpec(
        name="fund_intraday",
        grain="fund_minute_or_tick",
        time_columns=("timestamp", "available_time"),
        primary_keys=("fund_code", "timestamp"),
        columns=(
            c("fund_code", "string", True, "基金代码"),
            c("timestamp", "datetime", True, "行情时间"),
            c("available_time", "datetime", True, "行情可见时间"),
            c("price", "float", True, "ETF/LOF 成交价或估值代理"),
            c("iopv", "float", False, "ETF IOPV"),
            c("premium_pct", "float", False, "溢折价百分比"),
            c("volume", "float", False, "成交量"),
            c("amount", "float", False, "成交额"),
            c("bid_ask_spread", "float", False, "买卖价差"),
        ),
    ),
    "index_daily": TableSpec(
        name="index_daily",
        grain="index_day",
        time_columns=("trade_date", "available_time"),
        primary_keys=("index_code", "trade_date"),
        columns=(
            c("index_code", "string", True, "指数代码"),
            c("trade_date", "datetime", True, "交易日"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("open", "float", True, "开盘价"),
            c("high", "float", True, "最高价"),
            c("low", "float", True, "最低价"),
            c("close", "float", True, "收盘价"),
            c("volume", "float", False, "成交量"),
            c("amount", "float", False, "成交额"),
            c("valuation", "float", False, "估值指标或估值分位"),
        ),
    ),
    "index_intraday": TableSpec(
        name="index_intraday",
        grain="index_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("index_code", "timestamp"),
        columns=(
            c("index_code", "string", True, "指数代码"),
            c("timestamp", "datetime", True, "行情时间"),
            c("available_time", "datetime", True, "行情可见时间"),
            c("price", "float", True, "指数点位"),
            c("return", "float", False, "分钟收益率"),
            c("volume", "float", False, "成交量"),
            c("amount", "float", False, "成交额"),
        ),
    ),
    "index_constituent": TableSpec(
        name="index_constituent",
        grain="index_constituent_effective_date",
        time_columns=("effective_date", "available_time"),
        primary_keys=("index_code", "stock_code", "effective_date"),
        columns=(
            c("index_code", "string", True, "指数代码"),
            c("stock_code", "string", True, "成分股代码"),
            c("effective_date", "datetime", True, "权重生效日期"),
            c("available_time", "datetime", True, "权重数据可见时间"),
            c("weight", "float", True, "成分股权重"),
            c("industry", "string", False, "行业分类"),
            c("free_float_mktcap", "float", False, "自由流通市值"),
        ),
    ),
    "stock_daily_intraday": TableSpec(
        name="stock_daily_intraday",
        grain="stock_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("stock_code", "timestamp"),
        columns=(
            c("stock_code", "string", True, "股票代码"),
            c("timestamp", "datetime", True, "日频或分钟时间"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("return", "float", True, "收益率"),
            c("volume", "float", False, "成交量"),
            c("turnover", "float", False, "换手率"),
            c("limit_status", "string", False, "涨停/跌停/正常"),
            c("northbound_holding", "float", False, "北向持仓"),
        ),
    ),
    "futures_bar": TableSpec(
        name="futures_bar",
        grain="futures_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("contract", "timestamp"),
        columns=(
            c("contract", "string", True, "期货合约"),
            c("underlying", "string", True, "标的，如 IF/IH/IC/IM/HSI/Brent"),
            c("timestamp", "datetime", True, "行情时间"),
            c("available_time", "datetime", True, "行情可见时间"),
            c("price", "float", True, "合约价格"),
            c("basis", "float", False, "基差"),
            c("open_interest", "float", False, "持仓量"),
            c("term_structure", "float", False, "期限结构斜率或价差"),
        ),
    ),
    "commodity_bar": TableSpec(
        name="commodity_bar",
        grain="commodity_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("symbol", "timestamp"),
        columns=(
            c("symbol", "string", True, "商品或合约代码"),
            c("asset_class", "string", True, "能源/金属/黑色/贵金属/农产品"),
            c("timestamp", "datetime", True, "行情时间"),
            c("available_time", "datetime", True, "行情可见时间"),
            c("price", "float", True, "价格"),
            c("return", "float", False, "收益率"),
            c("volatility", "float", False, "波动率"),
        ),
    ),
    "option_volatility": TableSpec(
        name="option_volatility",
        grain="underlying_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("underlying", "timestamp"),
        columns=(
            c("underlying", "string", True, "期权标的"),
            c("timestamp", "datetime", True, "时间"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("iv", "float", True, "隐含波动率"),
            c("skew", "float", False, "波动率偏斜"),
            c("put_call_ratio", "float", False, "看跌/看涨比"),
            c("volume", "float", False, "成交量"),
            c("open_interest", "float", False, "持仓量"),
        ),
    ),
    "macro_rate_fx": TableSpec(
        name="macro_rate_fx",
        grain="macro_observation",
        time_columns=("timestamp", "release_time", "available_time"),
        primary_keys=("symbol", "timestamp"),
        columns=(
            c("symbol", "string", True, "宏观、利率或汇率指标代码"),
            c("timestamp", "datetime", True, "指标观察时间"),
            c("release_time", "datetime", True, "官方发布时间"),
            c("available_time", "datetime", True, "系统可见时间"),
            c("value", "float", True, "指标值"),
            c("change", "float", False, "变化值"),
        ),
    ),
    "cross_market": TableSpec(
        name="cross_market",
        grain="market_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("market", "timestamp"),
        columns=(
            c("market", "string", True, "US/HK/JP/KR/EU/GLOBAL"),
            c("timestamp", "datetime", True, "市场时间"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("index_return", "float", True, "市场指数收益"),
            c("vix", "float", False, "波动率指数或替代指标"),
            c("risk_on_off", "float", False, "风险偏好因子"),
        ),
    ),
    "capital_flow": TableSpec(
        name="capital_flow",
        grain="market_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("market", "timestamp"),
        columns=(
            c("market", "string", True, "CN/HK/GLOBAL"),
            c("timestamp", "datetime", True, "时间"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("northbound", "float", False, "北向资金"),
            c("southbound", "float", False, "南向资金"),
            c("etf_flow", "float", False, "ETF 资金流"),
            c("margin_balance", "float", False, "融资余额"),
        ),
    ),
    "sentiment_event": TableSpec(
        name="sentiment_event",
        grain="event",
        time_columns=("event_time", "release_time", "available_time"),
        primary_keys=("event_id",),
        columns=(
            c("event_id", "string", True, "事件唯一标识"),
            c("event_time", "datetime", True, "事件发生时间"),
            c("release_time", "datetime", True, "内容发布时间"),
            c("available_time", "datetime", True, "系统可见时间"),
            c("entity", "string", True, "关联指数/基金/股票/行业/市场"),
            c("topic", "string", True, "政策/监管/行业/地缘/流动性等主题"),
            c("sentiment_score", "float", True, "情绪分数 [-1, 1]"),
            c("panic_score", "float", False, "恐慌分数 [0, 1]"),
            c("source", "string", True, "新闻、公告或社媒来源"),
        ),
    ),
    "panic_factor": TableSpec(
        name="panic_factor",
        grain="market_day_or_minute",
        time_columns=("timestamp", "available_time"),
        primary_keys=("market", "timestamp"),
        columns=(
            c("market", "string", True, "CN/HK/GLOBAL"),
            c("timestamp", "datetime", True, "时间"),
            c("available_time", "datetime", True, "数据可见时间"),
            c("fear_score", "float", True, "合成恐慌分数"),
            c("iv_component", "float", False, "隐含波动率贡献"),
            c("flow_component", "float", False, "资金流贡献"),
            c("news_component", "float", False, "新闻舆情贡献"),
            c("limit_component", "float", False, "涨跌停或市场宽度贡献"),
        ),
    ),
    "label_daily_weekly": TableSpec(
        name="label_daily_weekly",
        grain="fund_day",
        time_columns=("trade_date", "label_available_time"),
        primary_keys=("fund_code", "trade_date"),
        columns=(
            c("fund_code", "string", True, "基金代码"),
            c("trade_date", "datetime", True, "预测起点交易日"),
            c("label_available_time", "datetime", True, "标签完整可见时间"),
            c("return_1d", "float", True, "未来一个交易日收益率"),
            c("return_1w", "float", True, "未来五个交易日收益率"),
            c("direction_1d", "string", True, "未来一个交易日方向"),
            c("direction_1w", "string", True, "未来一周方向"),
            c("tracking_error_1d", "float", False, "一日跟踪误差"),
            c("tracking_error_1w", "float", False, "一周跟踪误差"),
        ),
    ),
    "label_intraday": TableSpec(
        name="label_intraday",
        grain="fund_minute",
        time_columns=("timestamp", "label_available_time"),
        primary_keys=("fund_code", "timestamp"),
        columns=(
            c("fund_code", "string", True, "基金代码"),
            c("timestamp", "datetime", True, "预测起点时间"),
            c("label_available_time", "datetime", True, "标签完整可见时间"),
            c("return_3m", "float", True, "未来 3 分钟收益率"),
            c("return_5m", "float", True, "未来 5 分钟收益率"),
            c("direction_3m", "string", True, "未来 3 分钟方向"),
            c("direction_5m", "string", True, "未来 5 分钟方向"),
            c("proxy_intraday", "bool", True, "是否为场外基金估值代理标签"),
        ),
    ),
    "prediction_log": TableSpec(
        name="prediction_log",
        grain="prediction",
        time_columns=("asof_time", "created_at", "label_due_time"),
        primary_keys=("prediction_id",),
        columns=(
            c("prediction_id", "string", True, "预测记录唯一标识"),
            c("fund_code", "string", True, "基金代码"),
            c("horizon", "string", True, "daily/weekly/intraday_3m/intraday_5m"),
            c("asof_time", "datetime", True, "预测时点"),
            c("created_at", "datetime", True, "预测生成时间"),
            c("model_version", "string", True, "模型版本"),
            c("feature_snapshot_id", "string", True, "特征快照标识"),
            c("predicted_return", "float", True, "预测收益率"),
            c("predicted_direction", "string", True, "预测方向"),
            c("confidence", "float", True, "置信度"),
            c("is_actionable", "bool", True, "是否可行动"),
            c("label_due_time", "datetime", True, "标签应回填时间"),
            c("actual_return", "float", False, "真实收益率，回填"),
            c("actual_direction", "string", False, "真实方向，回填"),
        ),
    ),
}


def known_tables() -> tuple[str, ...]:
    return tuple(sorted(TABLE_SPECS))


def get_table_spec(table_name: str) -> TableSpec:
    try:
        return TABLE_SPECS[table_name]
    except KeyError as exc:
        raise ValueError(f"Unknown table '{table_name}'. Known tables: {', '.join(known_tables())}") from exc


def validate_columns(table_name: str, columns: Iterable[str]) -> list[str]:
    spec = get_table_spec(table_name)
    present = set(columns)
    return [name for name in spec.required_columns if name not in present]


def validate_frame(table_name: str, df: Any) -> list[str]:
    spec = get_table_spec(table_name)
    errors: list[str] = []
    missing = validate_columns(table_name, df.columns)
    if missing:
        errors.append(f"Missing required columns: {', '.join(missing)}")

    try:
        import pandas as pd
    except ImportError as exc:
        raise SystemExit("Missing dependency: pandas. Run `pip install -r requirements.txt`.") from exc

    for column in spec.time_columns:
        if column not in df.columns:
            continue
        raw_values = df[column]
        non_empty = raw_values.notna() & (raw_values.astype(str).str.strip() != "")
        parsed = pd.to_datetime(df[column], errors="coerce")
        invalid = non_empty & parsed.isna()
        if invalid.any():
            bad_count = int(invalid.sum())
            errors.append(f"Column '{column}' has {bad_count} invalid datetime value(s).")

    if "available_time" in df.columns:
        available = pd.to_datetime(df["available_time"], errors="coerce")
        for event_col in ("timestamp", "trade_date", "release_time", "event_time"):
            if event_col not in df.columns:
                continue
            event_time = pd.to_datetime(df[event_col], errors="coerce")
            invalid = available.notna() & event_time.notna() & (available < event_time)
            if invalid.any():
                errors.append(
                    f"Column 'available_time' is earlier than '{event_col}' in {int(invalid.sum())} row(s)."
                )

    return errors


def data_dictionary() -> dict[str, Any]:
    return {
        name: {
            **asdict(spec),
            "required_columns": list(spec.required_columns),
        }
        for name, spec in sorted(TABLE_SPECS.items())
    }
