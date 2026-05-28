from __future__ import annotations

from typing import Any

import numpy as np
from sklearn.metrics import accuracy_score, balanced_accuracy_score, classification_report, confusion_matrix

from .schema import ID_TO_LABEL


def build_report(y_true, y_pred, probabilities, accuracy_target: float = 0.98) -> dict[str, Any]:
    report: dict[str, Any] = {
        "accuracy": float(accuracy_score(y_true, y_pred)),
        "balanced_accuracy": float(balanced_accuracy_score(y_true, y_pred)),
        "classification_report": classification_report(
            y_true,
            y_pred,
            labels=[0, 1, 2],
            target_names=[ID_TO_LABEL[0], ID_TO_LABEL[1], ID_TO_LABEL[2]],
            output_dict=True,
            zero_division=0,
        ),
        "confusion_matrix": confusion_matrix(y_true, y_pred, labels=[0, 1, 2]).tolist(),
    }

    if probabilities is not None and len(probabilities) > 0:
        max_prob = np.max(probabilities, axis=1)
        high_conf_mask = max_prob >= accuracy_target
        coverage = float(np.mean(high_conf_mask))
        report["high_confidence_threshold"] = accuracy_target
        report["high_confidence_coverage"] = coverage
        if np.any(high_conf_mask):
            report["high_confidence_accuracy"] = float(accuracy_score(
                np.asarray(y_true)[high_conf_mask],
                np.asarray(y_pred)[high_conf_mask],
            ))
        else:
            report["high_confidence_accuracy"] = None
        report["calibration"] = probability_calibration_report(y_true, y_pred, probabilities)
    return report


def probability_calibration_report(
    y_true,
    y_pred,
    probabilities,
    bins: int = 10,
    class_count: int = 3,
) -> dict[str, Any]:
    if probabilities is None or len(probabilities) == 0:
        return {
            "ece": None,
            "mce": None,
            "brier_score": None,
            "bins": [],
        }

    probs = np.asarray(probabilities, dtype="float64")
    if probs.ndim != 2 or probs.shape[0] == 0:
        return {
            "ece": None,
            "mce": None,
            "brier_score": None,
            "bins": [],
        }

    y_true_arr = np.asarray(y_true, dtype="int64")
    y_pred_arr = np.asarray(y_pred)
    confidences = np.max(probs, axis=1)
    correct = (y_pred_arr == y_true_arr).astype("float64")

    edges = np.linspace(0.0, 1.0, bins + 1)
    ece = 0.0
    mce = 0.0
    bin_reports: list[dict[str, Any]] = []
    for idx in range(bins):
        left = edges[idx]
        right = edges[idx + 1]
        if idx == bins - 1:
            mask = (confidences >= left) & (confidences <= right)
        else:
            mask = (confidences >= left) & (confidences < right)
        count = int(np.sum(mask))
        if count == 0:
            bin_reports.append({
                "lower": round(float(left), 4),
                "upper": round(float(right), 4),
                "count": 0,
                "accuracy": None,
                "confidence": None,
                "gap": None,
            })
            continue
        accuracy = float(np.mean(correct[mask]))
        confidence = float(np.mean(confidences[mask]))
        gap = abs(accuracy - confidence)
        weight = count / len(confidences)
        ece += weight * gap
        mce = max(mce, gap)
        bin_reports.append({
            "lower": round(float(left), 4),
            "upper": round(float(right), 4),
            "count": count,
            "accuracy": round(accuracy, 6),
            "confidence": round(confidence, 6),
            "gap": round(gap, 6),
        })

    brier_score = None
    if probs.shape[1] == class_count:
        target = np.zeros((len(y_true_arr), class_count), dtype="float64")
        valid = (y_true_arr >= 0) & (y_true_arr < class_count)
        target[np.arange(len(y_true_arr))[valid], y_true_arr[valid].astype(int)] = 1.0
        brier_score = float(np.mean(np.sum((probs - target) ** 2, axis=1)))

    return {
        "ece": round(float(ece), 6),
        "mce": round(float(mce), 6),
        "brier_score": round(brier_score, 6) if brier_score is not None else None,
        "bins": bin_reports,
    }
