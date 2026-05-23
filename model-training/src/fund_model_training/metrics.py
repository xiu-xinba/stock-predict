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
    return report
