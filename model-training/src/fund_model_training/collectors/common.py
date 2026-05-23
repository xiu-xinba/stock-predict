from __future__ import annotations

from datetime import datetime
from pathlib import Path
from typing import Iterable

from fund_model_training.index_fund_contract import get_table_spec, validate_frame


def current_available_time() -> str:
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def first_existing(columns: Iterable[str], candidates: Iterable[str]) -> str | None:
    present = set(columns)
    return next((name for name in candidates if name in present), None)


def require_pandas():
    try:
        import pandas as pd
    except ImportError as exc:
        raise SystemExit("Missing dependency: pandas. Run `pip install -r requirements.txt`.") from exc
    return pd


def read_tracking_map(path: str | Path | None):
    pd = require_pandas()
    if not path:
        return {}
    df = pd.read_csv(path, dtype=str).fillna("")
    if "fund_code" not in df.columns or "tracking_index" not in df.columns:
        raise ValueError("Tracking map must contain 'fund_code' and 'tracking_index' columns.")
    mapping = {}
    for _, row in df.iterrows():
        mapping[str(row["fund_code"]).zfill(6)] = {
            "tracking_index": str(row["tracking_index"]).strip() or "UNMAPPED",
            "market": str(row.get("market", "")).strip() or None,
        }
    return mapping


def standardize_code(value) -> str:
    text = str(value).strip()
    if text.endswith(".0"):
        text = text[:-2]
    return text.zfill(6) if text.isdigit() and len(text) <= 6 else text


def infer_market_from_name(name: str, fallback: str = "CN") -> str:
    text = str(name)
    hk_keywords = ("港", "恒生", "恒指", "国企", "H股", "HK", "Hong Kong")
    global_keywords = ("纳指", "标普", "道琼", "德国", "法国", "日经", "印度", "美元", "全球", "美股", "海外")
    if any(keyword in text for keyword in hk_keywords):
        return "HK"
    if any(keyword in text for keyword in global_keywords):
        return "GLOBAL"
    return fallback


def numeric_series(df, candidates: Iterable[str], default=None):
    pd = require_pandas()
    column = first_existing(df.columns, candidates)
    if column is None:
        return pd.Series(default, index=df.index)
    return pd.to_numeric(df[column], errors="coerce")


def text_series(df, candidates: Iterable[str], default: str = ""):
    pd = require_pandas()
    column = first_existing(df.columns, candidates)
    if column is None:
        return pd.Series(default, index=df.index, dtype="object")
    return df[column].astype(str)


def datetime_series(df, candidates: Iterable[str], default: str | None = None):
    pd = require_pandas()
    column = first_existing(df.columns, candidates)
    if column is None:
        return pd.Series(default or current_available_time(), index=df.index, dtype="object")
    return pd.to_datetime(df[column], errors="coerce").dt.strftime("%Y-%m-%d %H:%M:%S")


def enforce_contract(df, table: str, skip_validation: bool = False):
    spec = get_table_spec(table)
    ordered_columns = [column.name for column in spec.columns if column.name in df.columns]
    extra_columns = [column for column in df.columns if column not in ordered_columns]
    out = df[ordered_columns + extra_columns].copy()

    if skip_validation:
        return out
    errors = validate_frame(table, out)
    if errors:
        joined = "\n- ".join(errors)
        raise ValueError(f"{table} contract validation failed:\n- {joined}")
    return out


def write_csv(df, path: str | Path) -> Path:
    out = Path(path)
    out.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(out, index=False, encoding="utf-8-sig")
    return out
