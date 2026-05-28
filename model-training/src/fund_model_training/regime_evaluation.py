from __future__ import annotations

import warnings
from typing import Any

import numpy as np
import pandas as pd
from sklearn.metrics import accuracy_score, balanced_accuracy_score, mean_absolute_error


def market_regime_report(
    test_df: pd.DataFrame,
    y_true_cls,
    y_pred_cls,
    y_true_reg,
    y_pred_reg,
    probabilities=None,
    high_confidence_threshold: float = 0.60,
    min_slice_rows: int = 5,
) -> dict[str, Any]:
    """Evaluate model behavior across design-specified market regimes."""

    frame = test_df.reset_index(drop=True).copy()
    frame["_y_true_cls"] = np.asarray(y_true_cls)
    frame["_y_pred_cls"] = np.asarray(y_pred_cls)
    frame["_y_true_reg"] = np.asarray(y_true_reg, dtype="float64")
    frame["_y_pred_reg"] = np.asarray(y_pred_reg, dtype="float64")
    if probabilities is not None and len(probabilities) > 0:
        frame["_confidence"] = np.max(np.asarray(probabilities, dtype="float64"), axis=1)
    else:
        frame["_confidence"] = np.nan

    groups = {
        "market_trend": _trend_regimes(frame),
        "panic": _binary_regime(frame, ["fear_score", "panic_score"], high="high_panic", low="low_panic"),
        "volatility": _binary_regime(
            frame,
            ["index_volatility_20d", "index_volatility_15m", "fund_volatility_20d", "fund_volatility_15m"],
            high="high_volatility",
            low="low_volatility",
        ),
    }

    reports: dict[str, Any] = {}
    weak_slices: list[dict[str, Any]] = []
    for group_name, labels in groups.items():
        if labels is None:
            reports[group_name] = {"status": "skipped", "reason": "required regime signal is missing or constant"}
            continue
        frame[f"_regime_{group_name}"] = labels
        slice_reports = {}
        for regime in sorted(labels.dropna().unique()):
            mask = labels == regime
            metrics = _slice_metrics(frame.loc[mask], high_confidence_threshold)
            slice_reports[str(regime)] = metrics
            if metrics["rows"] >= min_slice_rows and _is_weak_slice(metrics):
                weak_slices.append({
                    "group": group_name,
                    "regime": str(regime),
                    "rows": metrics["rows"],
                    "classification_accuracy": metrics["classification_accuracy"],
                    "balanced_accuracy": metrics["balanced_accuracy"],
                    "high_confidence_accuracy": metrics["high_confidence"]["accuracy"],
                })
        reports[group_name] = {
            "status": "ok",
            "slices": slice_reports,
        }

    return {
        "min_slice_rows": int(min_slice_rows),
        "groups": reports,
        "weak_slices": weak_slices,
    }


def _slice_metrics(frame: pd.DataFrame, high_confidence_threshold: float) -> dict[str, Any]:
    y_true = frame["_y_true_cls"].to_numpy()
    y_pred = frame["_y_pred_cls"].to_numpy()
    confidence = pd.to_numeric(frame["_confidence"], errors="coerce")
    high_conf_mask = confidence >= high_confidence_threshold
    high_conf_accuracy = None
    if high_conf_mask.any():
        high_conf_accuracy = float(accuracy_score(y_true[high_conf_mask.to_numpy()], y_pred[high_conf_mask.to_numpy()]))

    return {
        "rows": int(len(frame)),
        "classification_accuracy": float(accuracy_score(y_true, y_pred)) if len(frame) else None,
        "balanced_accuracy": _safe_balanced_accuracy(y_true, y_pred),
        "regression_mae": float(mean_absolute_error(frame["_y_true_reg"], frame["_y_pred_reg"])) if len(frame) else None,
        "high_confidence": {
            "threshold": float(high_confidence_threshold),
            "coverage": float(high_conf_mask.mean()) if len(frame) else 0.0,
            "accuracy": high_conf_accuracy,
        },
    }


def _safe_balanced_accuracy(y_true, y_pred) -> float | None:
    if len(y_true) == 0:
        return None
    try:
        with warnings.catch_warnings():
            warnings.filterwarnings("ignore", message="y_pred contains classes not in y_true")
            warnings.filterwarnings("ignore", message="A single label was found in 'y_true' and 'y_pred'.*")
            return float(balanced_accuracy_score(y_true, y_pred))
    except ValueError:
        return None


def _trend_regimes(frame: pd.DataFrame) -> pd.Series | None:
    signal = _first_numeric(frame, ["index_return_5d", "index_return_5m", "index_return_3m", "index_return_1d", "index_return_1m", "market_change_pct"])
    if signal is None:
        return None
    if signal.nunique(dropna=True) < 2:
        return None
    low, high = signal.quantile([1 / 3, 2 / 3])
    if not np.isfinite(low) or not np.isfinite(high) or low == high:
        return None
    return pd.Series(
        np.select(
            [signal <= low, signal >= high],
            ["bear", "bull"],
            default="sideways",
        ),
        index=frame.index,
    )


def _binary_regime(frame: pd.DataFrame, columns: list[str], high: str, low: str) -> pd.Series | None:
    signal = _first_numeric(frame, columns)
    if signal is None:
        return None
    if signal.nunique(dropna=True) < 2:
        return None
    median = signal.median()
    if not np.isfinite(median):
        return None
    return pd.Series(np.where(signal >= median, high, low), index=frame.index)


def _first_numeric(frame: pd.DataFrame, columns: list[str]) -> pd.Series | None:
    for column in columns:
        if column in frame.columns:
            values = pd.to_numeric(frame[column], errors="coerce")
            if values.notna().any():
                return values.fillna(values.median())
    return None


def _is_weak_slice(metrics: dict[str, Any]) -> bool:
    accuracy = metrics.get("classification_accuracy")
    balanced = metrics.get("balanced_accuracy")
    high_conf_accuracy = (metrics.get("high_confidence") or {}).get("accuracy")
    return (
        (accuracy is not None and accuracy < 0.34) or
        (balanced is not None and balanced < 0.30) or
        (high_conf_accuracy is not None and high_conf_accuracy < 0.40)
    )
