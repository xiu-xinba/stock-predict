from __future__ import annotations

from typing import Any, Iterable

import numpy as np


DEFAULT_LEVEL = 0.90
DEFAULT_LEVELS = (0.80, 0.90)


def empirical_prediction_interval_report(
    y_true: Iterable[float],
    y_pred: Iterable[float],
    levels: Iterable[float] = DEFAULT_LEVELS,
) -> dict[str, Any]:
    """Build residual-quantile intervals from holdout predictions."""
    true = np.asarray(list(y_true), dtype="float64")
    pred = np.asarray(list(y_pred), dtype="float64")
    mask = np.isfinite(true) & np.isfinite(pred)
    residuals = true[mask] - pred[mask]
    if residuals.size == 0:
        return {
            "enabled": False,
            "method": "empirical_residual_quantile",
            "reason": "no finite residuals",
            "residual_count": 0,
        }

    interval_levels: dict[str, dict[str, Any]] = {}
    for raw_level in levels:
        level = float(raw_level)
        if level <= 0.0 or level >= 1.0:
            continue
        alpha = 1.0 - level
        lower_q = float(np.quantile(residuals, alpha / 2.0))
        upper_q = float(np.quantile(residuals, 1.0 - alpha / 2.0))
        covered = (residuals >= lower_q) & (residuals <= upper_q)
        interval_levels[_level_key(level)] = {
            "level": round(level, 6),
            "lower_residual_quantile": round(lower_q, 6),
            "upper_residual_quantile": round(upper_q, 6),
            "empirical_coverage": round(float(covered.mean()), 6),
            "mean_width": round(float(upper_q - lower_q), 6),
        }

    return {
        "enabled": bool(interval_levels),
        "method": "empirical_residual_quantile",
        "default_level": DEFAULT_LEVEL,
        "residual_count": int(residuals.size),
        "residual_mean": round(float(residuals.mean()), 6),
        "residual_mae": round(float(np.mean(np.abs(residuals))), 6),
        "levels": interval_levels,
    }


def interval_bounds(
    predicted_return: float,
    report: dict[str, Any] | None,
    fallback_spread: float,
    level: float = DEFAULT_LEVEL,
) -> dict[str, Any]:
    if not report or not report.get("enabled"):
        return _fallback_bounds(predicted_return, fallback_spread)
    levels = report.get("levels") or {}
    selected = levels.get(_level_key(level))
    if not selected:
        selected = _nearest_level(levels, level)
    if not selected:
        return _fallback_bounds(predicted_return, fallback_spread)

    lower_residual = _float_or_none(selected.get("lower_residual_quantile"))
    upper_residual = _float_or_none(selected.get("upper_residual_quantile"))
    if lower_residual is None or upper_residual is None:
        return _fallback_bounds(predicted_return, fallback_spread)
    low = float(predicted_return) + lower_residual
    high = float(predicted_return) + upper_residual
    return {
        "low": round(float(min(low, high)), 4),
        "high": round(float(max(low, high)), 4),
        "method": "empirical_residual_quantile",
        "level": selected.get("level", level),
        "empirical_coverage": selected.get("empirical_coverage"),
    }


def _fallback_bounds(predicted_return: float, fallback_spread: float) -> dict[str, Any]:
    return {
        "low": round(float(predicted_return) - float(fallback_spread), 4),
        "high": round(float(predicted_return) + float(fallback_spread), 4),
        "method": "heuristic_spread",
        "level": None,
        "empirical_coverage": None,
    }


def _nearest_level(levels: dict[str, Any], target: float) -> dict[str, Any] | None:
    candidates = []
    for value in levels.values():
        level = _float_or_none((value or {}).get("level"))
        if level is not None:
            candidates.append((abs(level - target), value))
    if not candidates:
        return None
    return sorted(candidates, key=lambda item: item[0])[0][1]


def _level_key(level: float) -> str:
    return f"{float(level):.2f}"


def _float_or_none(value: Any) -> float | None:
    try:
        if value is None:
            return None
        numeric = float(value)
    except (TypeError, ValueError):
        return None
    if not np.isfinite(numeric):
        return None
    return numeric
