from __future__ import annotations

import argparse
import json
from pathlib import Path

from fund_model_training.collectors.common import current_available_time, enforce_contract, require_pandas, write_csv


DEFAULT_WEIGHTS = {
    "iv_component": 0.35,
    "flow_component": 0.25,
    "news_component": 0.25,
    "limit_component": 0.15,
}


def main() -> None:
    parser = argparse.ArgumentParser(description="Build point-in-time panic_factor rows from component CSVs.")
    parser.add_argument("--input", type=Path, required=True, help="Input CSV with market/timestamp and component columns.")
    parser.add_argument("--output", type=Path, required=True, help="Output panic_factor CSV.")
    parser.add_argument("--market", help="Market value if input CSV does not contain a market column.")
    parser.add_argument("--timestamp-col", default="timestamp")
    parser.add_argument("--available-time-col", default="available_time")
    parser.add_argument("--weights", help="Comma list, e.g. iv_component=0.4,flow_component=0.2")
    parser.add_argument("--skip-validation", action="store_true")
    args = parser.parse_args()

    weights = {**DEFAULT_WEIGHTS, **_parse_weights(args.weights)}
    df = build_panic_factor(
        input_path=args.input,
        market=args.market,
        timestamp_col=args.timestamp_col,
        available_time_col=args.available_time_col,
        weights=weights,
        skip_validation=args.skip_validation,
    )
    out = write_csv(df, args.output)
    print(json.dumps({"ok": True, "rows": int(len(df)), "output": str(out)}, ensure_ascii=False, indent=2))


def build_panic_factor(
    input_path: str | Path,
    market: str | None,
    timestamp_col: str,
    available_time_col: str,
    weights: dict[str, float],
    skip_validation: bool = False,
):
    pd = require_pandas()
    raw = pd.read_csv(input_path)
    return build_panic_factor_from_frame(
        raw=raw,
        market=market,
        timestamp_col=timestamp_col,
        available_time_col=available_time_col,
        weights=weights,
        skip_validation=skip_validation,
    )


def build_panic_factor_from_frame(
    raw,
    market: str | None,
    timestamp_col: str,
    available_time_col: str,
    weights: dict[str, float],
    skip_validation: bool = False,
):
    pd = require_pandas()
    if timestamp_col not in raw.columns:
        raise ValueError(f"Input CSV missing timestamp column: {timestamp_col}")

    out = pd.DataFrame()
    out["market"] = raw["market"].astype(str) if "market" in raw.columns else market
    if out["market"].isna().any() or (out["market"].astype(str).str.strip() == "").any():
        raise ValueError("Provide --market or include a non-empty market column.")
    out["timestamp"] = pd.to_datetime(raw[timestamp_col], errors="coerce").dt.strftime("%Y-%m-%d %H:%M:%S")
    if available_time_col in raw.columns:
        out["available_time"] = pd.to_datetime(raw[available_time_col], errors="coerce").dt.strftime("%Y-%m-%d %H:%M:%S")
    else:
        out["available_time"] = current_available_time()

    normalized_components = {}
    for component in DEFAULT_WEIGHTS:
        source = pd.to_numeric(raw[component], errors="coerce") if component in raw.columns else pd.Series(0.0, index=raw.index)
        normalized = source.groupby(out["market"], group_keys=False).transform(_expanding_percentile)
        out[component] = normalized.fillna(0.0)
        normalized_components[component] = out[component]

    total_weight = sum(abs(weight) for weight in weights.values()) or 1.0
    fear = pd.Series(0.0, index=out.index)
    for component, weight in weights.items():
        if component not in normalized_components:
            continue
        fear = fear + normalized_components[component] * weight
    out["fear_score"] = (fear / total_weight).clip(0.0, 1.0)

    return enforce_contract(out, "panic_factor", skip_validation=skip_validation)


def _expanding_percentile(values):
    pd = require_pandas()
    numeric = pd.to_numeric(values, errors="coerce")
    result = []
    history: list[float] = []
    for value in numeric:
        if pd.isna(value):
            result.append(None)
            continue
        history.append(float(value))
        rank = sum(item <= float(value) for item in history) / len(history)
        result.append(rank)
    return pd.Series(result, index=values.index)


def _parse_weights(raw: str | None) -> dict[str, float]:
    if not raw:
        return {}
    parsed: dict[str, float] = {}
    for chunk in raw.split(","):
        if not chunk.strip():
            continue
        name, _, value = chunk.partition("=")
        if not name or not value:
            raise ValueError(f"Invalid weight chunk: {chunk}")
        parsed[name.strip()] = float(value)
    return parsed


if __name__ == "__main__":
    main()
