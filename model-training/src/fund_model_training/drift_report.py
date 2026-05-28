from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import numpy as np

from fund_model_training.collectors.common import require_pandas
from fund_model_training.features import prepare_features


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate a lightweight feature drift report for model samples.")
    parser.add_argument("--samples", type=Path, required=True, help="Processed sample CSV.")
    parser.add_argument("--feature-set", required=True, help="Feature set name, e.g. index_fund_daily_v1.")
    parser.add_argument("--output", type=Path, required=True, help="JSON report output path.")
    parser.add_argument("--reference-start", help="Optional inclusive reference window start.")
    parser.add_argument("--reference-end", help="Optional inclusive reference window end.")
    parser.add_argument("--current-start", help="Optional inclusive current window start.")
    parser.add_argument("--current-end", help="Optional inclusive current window end.")
    parser.add_argument("--reference-rows", type=int, default=1000)
    parser.add_argument("--current-rows", type=int, default=250)
    parser.add_argument("--psi-bins", type=int, default=10)
    parser.add_argument("--psi-threshold", type=float, default=0.20)
    parser.add_argument("--ks-threshold", type=float, default=0.20)
    parser.add_argument("--missing-delta-threshold", type=float, default=0.10)
    args = parser.parse_args()

    report = build_drift_report(
        samples_path=args.samples,
        feature_set=args.feature_set,
        output_path=args.output,
        reference_start=args.reference_start,
        reference_end=args.reference_end,
        current_start=args.current_start,
        current_end=args.current_end,
        reference_rows=args.reference_rows,
        current_rows=args.current_rows,
        psi_bins=args.psi_bins,
        psi_threshold=args.psi_threshold,
        ks_threshold=args.ks_threshold,
        missing_delta_threshold=args.missing_delta_threshold,
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))


def build_drift_report(
    samples_path: str | Path,
    feature_set: str,
    output_path: str | Path,
    reference_start: str | None = None,
    reference_end: str | None = None,
    current_start: str | None = None,
    current_end: str | None = None,
    reference_rows: int = 1000,
    current_rows: int = 250,
    psi_bins: int = 10,
    psi_threshold: float = 0.20,
    ks_threshold: float = 0.20,
    missing_delta_threshold: float = 0.10,
) -> dict[str, Any]:
    pd = require_pandas()
    samples = pd.read_csv(samples_path, dtype={"fund_code": str})
    prepared, feature_names = prepare_features(samples, feature_set)
    prepared = prepared.sort_values("asof_time").reset_index(drop=True)

    reference = _select_window(
        prepared,
        start=reference_start,
        end=reference_end,
        rows=reference_rows,
        prefer_tail=True,
        exclude_tail_rows=current_rows if not current_start and not current_end else 0,
    )
    current = _select_window(
        prepared,
        start=current_start,
        end=current_end,
        rows=current_rows,
        prefer_tail=True,
    )
    if reference.empty:
        raise ValueError("Reference drift window is empty.")
    if current.empty:
        raise ValueError("Current drift window is empty.")

    features = []
    for name in feature_names:
        block = _feature_drift_block(
            name=name,
            reference=reference[name],
            current=current[name],
            psi_bins=psi_bins,
            psi_threshold=psi_threshold,
            ks_threshold=ks_threshold,
            missing_delta_threshold=missing_delta_threshold,
        )
        features.append(block)

    drifted = [item for item in features if item["drifted"]]
    max_psi = _max_feature(features, "psi")
    max_ks = _max_feature(features, "ks_statistic")
    report = {
        "ok": True,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "samples_path": str(samples_path),
        "feature_set": feature_set,
        "rows": int(len(prepared)),
        "reference": _window_summary(reference),
        "current": _window_summary(current),
        "thresholds": {
            "psi": psi_threshold,
            "ks": ks_threshold,
            "missing_delta": missing_delta_threshold,
        },
        "drift_detected": bool(drifted),
        "drifted_features": int(len(drifted)),
        "max_psi_feature": max_psi,
        "max_ks_feature": max_ks,
        "features": features,
    }
    output = Path(output_path)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    return report


def population_stability_index(reference: Any, current: Any, bins: int = 10) -> float | None:
    ref = _clean_array(reference)
    cur = _clean_array(current)
    if len(ref) == 0 or len(cur) == 0:
        return None
    edges = _psi_edges(ref, cur, bins)
    if edges is None:
        return 0.0
    ref_counts, _ = np.histogram(ref, bins=edges)
    cur_counts, _ = np.histogram(cur, bins=edges)
    epsilon = 1e-6
    ref_pct = (ref_counts + epsilon) / (len(ref) + epsilon * len(ref_counts))
    cur_pct = (cur_counts + epsilon) / (len(cur) + epsilon * len(cur_counts))
    psi = np.sum((cur_pct - ref_pct) * np.log(cur_pct / ref_pct))
    return round(float(psi), 6)


def ks_statistic(reference: Any, current: Any) -> float | None:
    ref = np.sort(_clean_array(reference))
    cur = np.sort(_clean_array(current))
    if len(ref) == 0 or len(cur) == 0:
        return None
    points = np.sort(np.unique(np.concatenate([ref, cur])))
    ref_cdf = np.searchsorted(ref, points, side="right") / len(ref)
    cur_cdf = np.searchsorted(cur, points, side="right") / len(cur)
    return round(float(np.max(np.abs(ref_cdf - cur_cdf))), 6)


def _feature_drift_block(
    name: str,
    reference: Any,
    current: Any,
    psi_bins: int,
    psi_threshold: float,
    ks_threshold: float,
    missing_delta_threshold: float,
) -> dict[str, Any]:
    ref = _clean_array(reference)
    cur = _clean_array(current)
    ref_missing = _missing_rate(reference)
    cur_missing = _missing_rate(current)
    psi = population_stability_index(ref, cur, bins=psi_bins)
    ks = ks_statistic(ref, cur)
    ref_mean = _round_or_none(np.mean(ref) if len(ref) else None)
    cur_mean = _round_or_none(np.mean(cur) if len(cur) else None)
    ref_std = _round_or_none(np.std(ref) if len(ref) else None)
    cur_std = _round_or_none(np.std(cur) if len(cur) else None)
    missing_delta = abs(cur_missing - ref_missing)
    drifted = (
        (psi is not None and psi >= psi_threshold) or
        (ks is not None and ks >= ks_threshold) or
        missing_delta >= missing_delta_threshold
    )
    return {
        "feature": name,
        "drifted": bool(drifted),
        "psi": psi,
        "ks_statistic": ks,
        "reference_missing_rate": round(float(ref_missing), 6),
        "current_missing_rate": round(float(cur_missing), 6),
        "missing_delta": round(float(missing_delta), 6),
        "reference_mean": ref_mean,
        "current_mean": cur_mean,
        "reference_std": ref_std,
        "current_std": cur_std,
    }


def _select_window(
    df: Any,
    start: str | None,
    end: str | None,
    rows: int,
    prefer_tail: bool,
    exclude_tail_rows: int = 0,
):
    pd = require_pandas()
    selected = df
    if exclude_tail_rows > 0 and len(selected) > exclude_tail_rows:
        selected = selected.iloc[:-exclude_tail_rows]
    if start:
        selected = selected.loc[selected["asof_time"] >= pd.to_datetime(start)]
    if end:
        selected = selected.loc[selected["asof_time"] <= pd.to_datetime(end)]
    if rows > 0 and len(selected) > rows:
        selected = selected.tail(rows) if prefer_tail else selected.head(rows)
    return selected.copy()


def _window_summary(df: Any) -> dict[str, Any]:
    start = df["asof_time"].min()
    end = df["asof_time"].max()
    return {
        "rows": int(len(df)),
        "start": start.isoformat() if hasattr(start, "isoformat") else str(start),
        "end": end.isoformat() if hasattr(end, "isoformat") else str(end),
        "funds": int(df["fund_code"].nunique()) if "fund_code" in df.columns else None,
    }


def _max_feature(features: list[dict[str, Any]], key: str) -> dict[str, Any] | None:
    valid = [item for item in features if item.get(key) is not None]
    if not valid:
        return None
    winner = max(valid, key=lambda item: float(item[key]))
    return {"feature": winner["feature"], key: winner[key]}


def _psi_edges(reference: np.ndarray, current: np.ndarray, bins: int) -> np.ndarray | None:
    bins = max(2, int(bins))
    edges = np.unique(np.quantile(reference, np.linspace(0, 1, bins + 1)))
    if len(edges) < 2:
        lower = float(min(np.min(reference), np.min(current)))
        upper = float(max(np.max(reference), np.max(current)))
        if lower == upper:
            return None
        edges = np.linspace(lower, upper, bins + 1)
    edges = edges.astype(float)
    edges[0] = -np.inf
    edges[-1] = np.inf
    return edges


def _clean_array(values: Any) -> np.ndarray:
    pd = require_pandas()
    series = pd.Series(values)
    series = pd.to_numeric(series, errors="coerce")
    return series.replace([np.inf, -np.inf], np.nan).dropna().to_numpy(dtype=float)


def _missing_rate(values: Any) -> float:
    pd = require_pandas()
    series = pd.Series(values)
    series = pd.to_numeric(series, errors="coerce").replace([np.inf, -np.inf], np.nan)
    if len(series) == 0:
        return 0.0
    return float(series.isna().mean())


def _round_or_none(value: Any) -> float | None:
    if value is None:
        return None
    try:
        if np.isnan(value):
            return None
    except TypeError:
        return None
    return round(float(value), 6)


if __name__ == "__main__":
    main()
