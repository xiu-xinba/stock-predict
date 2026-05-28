from __future__ import annotations

import argparse
import hashlib
import json
from datetime import timedelta
from pathlib import Path
from typing import Any

from fund_model_training.collectors.common import enforce_contract, require_pandas, standardize_code, write_csv


HORIZON_RETURN_COLUMNS = {
    "next_day": "future_return_pct_next_day",
    "next_week": "future_return_pct_1w",
    "intraday_3m": "future_return_pct_3m",
    "intraday_5m": "future_return_pct_5m",
}


def main() -> None:
    parser = argparse.ArgumentParser(description="Create and backfill prediction_log rows.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    log_parser = subparsers.add_parser("log", help="Append a model prediction JSON to prediction_log CSV.")
    log_parser.add_argument("--prediction-json", type=Path, required=True)
    log_parser.add_argument("--output", type=Path, required=True)
    log_parser.add_argument("--append", action="store_true")

    backfill_parser = subparsers.add_parser("backfill", help="Backfill actual labels from processed samples.")
    backfill_parser.add_argument("--prediction-log", type=Path, required=True)
    backfill_parser.add_argument("--samples", type=Path, required=True)
    backfill_parser.add_argument("--output", type=Path, required=True)
    backfill_parser.add_argument("--flat-threshold-pct", type=float, default=0.02)
    backfill_parser.add_argument("--horizon", action="append", help="Only backfill matching horizon rows. May be supplied more than once.")

    eval_parser = subparsers.add_parser("evaluate", help="Summarize backfilled prediction performance.")
    eval_parser.add_argument("--prediction-log", type=Path, required=True)
    eval_parser.add_argument("--output", type=Path, required=True)
    eval_parser.add_argument("--high-confidence-threshold", type=float, default=0.60)
    eval_parser.add_argument("--round-trip-cost-pct", type=float, default=0.0)

    args = parser.parse_args()
    if args.command == "log":
        payload = json.loads(args.prediction_json.read_text(encoding="utf-8"))
        result = append_prediction_log(payload, args.output, append=args.append)
    elif args.command == "backfill":
        result = backfill_prediction_log(
            prediction_log_path=args.prediction_log,
            samples_path=args.samples,
            output_path=args.output,
            flat_threshold_pct=args.flat_threshold_pct,
            horizons=args.horizon,
        )
    elif args.command == "evaluate":
        result = evaluate_prediction_log(
            prediction_log_path=args.prediction_log,
            output_path=args.output,
            high_confidence_threshold=args.high_confidence_threshold,
            round_trip_cost_pct=args.round_trip_cost_pct,
        )
    else:  # pragma: no cover - argparse prevents this
        raise SystemExit(f"Unsupported command: {args.command}")
    print(json.dumps(result, ensure_ascii=False, indent=2))


def append_prediction_log(prediction_payload: dict[str, Any], output_path: str | Path, append: bool = True) -> dict[str, Any]:
    pd = require_pandas()
    row = prediction_payload_to_log_row(prediction_payload)
    new_df = pd.DataFrame([row])
    output = Path(output_path)
    if append and output.exists():
        old = pd.read_csv(output, dtype={"fund_code": str, "prediction_id": str})
        combined = pd.concat([old, new_df], ignore_index=True)
        combined = combined.drop_duplicates(subset=["prediction_id"], keep="last")
    else:
        combined = new_df
    ordered = enforce_contract(combined, "prediction_log", skip_validation=True)
    write_csv(ordered, output)
    return {"ok": True, "rows": int(len(ordered)), "prediction_id": row["prediction_id"], "output": str(output)}


def prediction_payload_to_log_row(payload: dict[str, Any]) -> dict[str, Any]:
    prediction = payload.get("prediction") or {}
    model = payload.get("model") or {}
    pd = require_pandas()
    fund_code = standardize_code(payload.get("fund_code", ""))
    horizon = str(prediction.get("horizon") or "next_day")
    asof_time = pd.to_datetime(payload.get("asof_time"), errors="coerce")
    if pd.isna(asof_time):
        raise ValueError("prediction payload missing valid asof_time")
    created_at = pd.to_datetime(payload.get("created_at"), errors="coerce")
    if pd.isna(created_at):
        created_at = pd.Timestamp.utcnow()
    model_version = _model_version(model)
    feature_snapshot = payload.get("feature_snapshot") or {}
    feature_snapshot_json = _stable_json(feature_snapshot) if feature_snapshot else ""
    feature_snapshot_id = _stable_id(
        "feature",
        fund_code,
        horizon,
        asof_time.isoformat(),
        feature_snapshot_json or model.get("feature_set", ""),
    )
    prediction_id = _stable_id("prediction", fund_code, horizon, asof_time.isoformat(), model_version)
    signal_status = _signal_status_from_prediction(prediction)
    return {
        "prediction_id": prediction_id,
        "fund_code": fund_code,
        "horizon": horizon,
        "asof_time": asof_time.strftime("%Y-%m-%d %H:%M:%S"),
        "created_at": created_at.strftime("%Y-%m-%d %H:%M:%S"),
        "model_version": model_version,
        "feature_snapshot_id": feature_snapshot_id,
        "feature_snapshot_json": feature_snapshot_json,
        "predicted_return": _float(prediction.get("predicted_change_pct")),
        "predicted_direction": str(prediction.get("direction") or "flat"),
        "confidence": _float(prediction.get("direction_confidence")),
        "signal_status": signal_status,
        "is_actionable": signal_status == "actionable",
        "label_due_time": _label_due_time(asof_time, horizon).strftime("%Y-%m-%d %H:%M:%S"),
        "actual_return": None,
        "actual_direction": None,
    }


def backfill_prediction_log(
    prediction_log_path: str | Path,
    samples_path: str | Path,
    output_path: str | Path,
    flat_threshold_pct: float = 0.02,
    horizons: list[str] | None = None,
) -> dict[str, Any]:
    pd = require_pandas()
    logs = pd.read_csv(prediction_log_path, dtype={"fund_code": str, "prediction_id": str})
    if horizons:
        allowed = {str(horizon) for horizon in horizons}
        logs = logs.loc[logs["horizon"].astype(str).isin(allowed)].copy()
    samples = pd.read_csv(samples_path, dtype={"fund_code": str})
    logs["fund_code"] = logs["fund_code"].map(standardize_code)
    samples["fund_code"] = samples["fund_code"].map(standardize_code)
    logs["asof_time"] = pd.to_datetime(logs["asof_time"], errors="coerce")
    samples["asof_time"] = pd.to_datetime(samples["asof_time"], errors="coerce")

    filled = logs.copy()
    filled = _ensure_signal_status_column(filled)
    if "actual_direction" not in filled.columns:
        filled["actual_direction"] = None
    filled["actual_direction"] = filled["actual_direction"].astype("object")
    if "actual_return" not in filled.columns:
        filled["actual_return"] = None
    filled_count = 0
    for idx, row in filled.iterrows():
        horizon = str(row.get("horizon", ""))
        return_col = HORIZON_RETURN_COLUMNS.get(horizon)
        if not return_col or return_col not in samples.columns:
            continue
        match = samples.loc[
            (samples["fund_code"] == row["fund_code"]) &
            (samples["asof_time"] == row["asof_time"])
        ]
        if match.empty:
            continue
        actual_return = _float(match.iloc[-1][return_col])
        if actual_return is None:
            continue
        filled.at[idx, "actual_return"] = actual_return
        filled.at[idx, "actual_direction"] = _direction_from_return(actual_return, flat_threshold_pct)
        filled_count += 1

    filled["hit_direction"] = filled.apply(
        lambda row: bool(row.get("predicted_direction") == row.get("actual_direction"))
        if str(row.get("actual_direction", "")).strip() else None,
        axis=1,
    )
    filled["absolute_error_pct"] = (
        pd.to_numeric(filled.get("predicted_return"), errors="coerce") -
        pd.to_numeric(filled.get("actual_return"), errors="coerce")
    ).abs()
    filled["asof_time"] = pd.to_datetime(filled["asof_time"], errors="coerce").dt.strftime("%Y-%m-%d %H:%M:%S")
    ordered = enforce_contract(filled, "prediction_log", skip_validation=True)
    write_csv(ordered, output_path)
    return {"ok": True, "rows": int(len(ordered)), "filled_rows": int(filled_count), "output": str(output_path)}


def evaluate_prediction_log(
    prediction_log_path: str | Path,
    output_path: str | Path,
    high_confidence_threshold: float = 0.60,
    round_trip_cost_pct: float = 0.0,
) -> dict[str, Any]:
    pd = require_pandas()
    df = pd.read_csv(prediction_log_path, dtype={"fund_code": str, "prediction_id": str})
    df = _ensure_signal_status_column(df)
    labeled = df.dropna(subset=["actual_direction", "actual_return"]).copy()
    labeled["hit_direction"] = labeled["predicted_direction"].astype(str) == labeled["actual_direction"].astype(str)
    labeled["absolute_error_pct"] = (
        pd.to_numeric(labeled["predicted_return"], errors="coerce") -
        pd.to_numeric(labeled["actual_return"], errors="coerce")
    ).abs()

    summary = {
        "ok": True,
        "rows": int(len(df)),
        "labeled_rows": int(len(labeled)),
        "overall": _performance_block(labeled, high_confidence_threshold, round_trip_cost_pct),
        "by_horizon": {},
    }
    for horizon, group in labeled.groupby("horizon"):
        summary["by_horizon"][str(horizon)] = _performance_block(group, high_confidence_threshold, round_trip_cost_pct)
    output = Path(output_path)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return summary


def _performance_block(df, high_confidence_threshold: float, round_trip_cost_pct: float) -> dict[str, Any]:
    pd = require_pandas()
    df = _ensure_signal_status_column(df.copy())
    if df.empty:
        return {
            "rows": 0,
            "direction_accuracy": None,
            "mae": None,
            "rmse": None,
            "high_confidence_coverage": 0.0,
            "high_confidence_accuracy": None,
            "actionable_coverage": 0.0,
            "actionable_accuracy": None,
            "signal_status": {
                "counts": {"actionable": 0, "low_confidence": 0, "no_signal": 0},
                "coverage": {"actionable": 0.0, "low_confidence": 0.0, "no_signal": 0.0},
            },
            "paper_trading": {
                "round_trip_cost_pct": round_trip_cost_pct,
                "mean_cost_adjusted_return": None,
                "cumulative_cost_adjusted_return": None,
                "win_rate_after_cost": None,
                "high_confidence_mean_return": None,
                "actionable_mean_return": None,
            },
        }
    errors = pd.to_numeric(df["absolute_error_pct"], errors="coerce")
    high_conf = df.loc[pd.to_numeric(df["confidence"], errors="coerce") >= high_confidence_threshold]
    actionable = df.loc[df["is_actionable"].astype(str).str.lower().isin(["true", "1"])]
    strategy_returns = _cost_adjusted_returns(df, round_trip_cost_pct)
    high_conf_returns = strategy_returns.loc[high_conf.index]
    actionable_returns = strategy_returns.loc[actionable.index]
    signal_status = df["signal_status"].astype(str).fillna("")
    signal_counts = {name: int((signal_status == name).sum()) for name in ("actionable", "low_confidence", "no_signal")}
    return {
        "rows": int(len(df)),
        "direction_accuracy": round(float(df["hit_direction"].mean()), 6),
        "mae": round(float(errors.mean()), 6),
        "rmse": round(float((errors.pow(2).mean()) ** 0.5), 6),
        "high_confidence_coverage": round(float(len(high_conf) / len(df)), 6),
        "high_confidence_accuracy": _accuracy_or_none(high_conf),
        "actionable_coverage": round(float(len(actionable) / len(df)), 6),
        "actionable_accuracy": _accuracy_or_none(actionable),
        "signal_status": {
            "counts": signal_counts,
            "coverage": {name: round(float(count / len(df)), 6) for name, count in signal_counts.items()},
        },
        "paper_trading": {
            "round_trip_cost_pct": round_trip_cost_pct,
            "mean_cost_adjusted_return": _mean_or_none(strategy_returns),
            "cumulative_cost_adjusted_return": _sum_or_none(strategy_returns),
            "win_rate_after_cost": _positive_rate_or_none(strategy_returns),
            "high_confidence_mean_return": _mean_or_none(high_conf_returns),
            "actionable_mean_return": _mean_or_none(actionable_returns),
        },
    }


def _accuracy_or_none(df) -> float | None:
    if df.empty:
        return None
    return round(float(df["hit_direction"].mean()), 6)


def _cost_adjusted_returns(df, round_trip_cost_pct: float):
    pd = require_pandas()
    signal = df["predicted_direction"].map({"up": 1.0, "down": -1.0, "flat": 0.0}).fillna(0.0)
    actual = pd.to_numeric(df["actual_return"], errors="coerce").fillna(0.0)
    cost = signal.abs() * float(round_trip_cost_pct)
    return signal * actual - cost


def _mean_or_none(values) -> float | None:
    if len(values) == 0:
        return None
    return round(float(values.mean()), 6)


def _sum_or_none(values) -> float | None:
    if len(values) == 0:
        return None
    return round(float(values.sum()), 6)


def _positive_rate_or_none(values) -> float | None:
    if len(values) == 0:
        return None
    return round(float((values > 0).mean()), 6)


def _model_version(model: dict[str, Any]) -> str:
    raw = model.get("model_path") or model.get("candidate") or "unknown"
    candidate = str(model.get("candidate") or "unknown")
    path_name = Path(str(raw)).stem
    return f"{candidate}:{path_name}"


def _label_due_time(asof_time, horizon: str):
    if horizon == "intraday_3m":
        return asof_time + timedelta(minutes=3)
    if horizon == "intraday_5m":
        return asof_time + timedelta(minutes=5)
    if horizon == "next_week":
        return asof_time + timedelta(days=7)
    return asof_time + timedelta(days=1)


def _direction_from_return(value: float, flat_threshold_pct: float) -> str:
    if value > flat_threshold_pct:
        return "up"
    if value < -flat_threshold_pct:
        return "down"
    return "flat"


def _ensure_signal_status_column(df):
    if "signal_status" in df.columns:
        df["signal_status"] = df.apply(_signal_status_from_log_row, axis=1)
        return df
    out = df.copy()
    out["signal_status"] = out.apply(_signal_status_from_log_row, axis=1)
    return out


def _signal_status_from_prediction(prediction: dict[str, Any]) -> str:
    raw_status = str(prediction.get("signal_status") or "").strip()
    if raw_status in {"actionable", "low_confidence", "no_signal"}:
        return raw_status
    if bool(prediction.get("is_actionable", False)):
        return "actionable"
    if str(prediction.get("direction") or "flat") == "flat":
        return "no_signal"
    return "low_confidence"


def _signal_status_from_log_row(row) -> str:
    raw_status = str(row.get("signal_status") or "").strip()
    if raw_status in {"actionable", "low_confidence", "no_signal"}:
        return raw_status
    if str(row.get("is_actionable", "")).lower() in {"true", "1"}:
        return "actionable"
    if str(row.get("predicted_direction") or "flat") == "flat":
        return "no_signal"
    return "low_confidence"


def _stable_id(*parts: str) -> str:
    raw = "|".join(str(part) for part in parts)
    return hashlib.sha1(raw.encode("utf-8")).hexdigest()[:24]


def _stable_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def _float(value: Any) -> float | None:
    try:
        if value is None:
            return None
        return float(value)
    except (TypeError, ValueError):
        return None


if __name__ == "__main__":
    main()
