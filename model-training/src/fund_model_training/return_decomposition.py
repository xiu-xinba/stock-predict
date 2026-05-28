from __future__ import annotations

from typing import Any

import numpy as np


TARGET_COMPONENTS = {
    "future_return_pct_next_day": (
        "future_index_return_pct_next_day",
        "future_tracking_error_pct_next_day",
    ),
    "future_return_pct_1w": (
        "future_index_return_pct_1w",
        "future_tracking_error_pct_1w",
    ),
    "future_return_pct_3m": (
        "future_index_return_pct_3m",
        "future_tracking_error_pct_3m",
    ),
    "future_return_pct_5m": (
        "future_index_return_pct_5m",
        "future_tracking_error_pct_5m",
    ),
}


def component_columns(regression_target: str) -> tuple[str, str] | None:
    return TARGET_COMPONENTS.get(regression_target)


def ensure_return_decomposition_targets(samples, regression_target: str) -> tuple[Any, list[str]]:
    components = component_columns(regression_target)
    if components is None:
        return samples, []

    index_col, tracking_error_col = components
    out = samples.copy()
    if index_col in out.columns and tracking_error_col not in out.columns:
        out[tracking_error_col] = _numeric(out[regression_target]) - _numeric(out[index_col])
    elif tracking_error_col in out.columns and index_col not in out.columns:
        out[index_col] = _numeric(out[regression_target]) - _numeric(out[tracking_error_col])

    if index_col in out.columns and tracking_error_col in out.columns:
        return out, [index_col, tracking_error_col]
    return out, []


def fit_return_decomposition(regressor, x_train, train_df, regression_target: str) -> dict[str, Any] | None:
    components = component_columns(regression_target)
    if components is None:
        return None
    index_col, tracking_error_col = components
    if index_col not in train_df.columns or tracking_error_col not in train_df.columns:
        return None

    try:
        from sklearn.base import clone
    except ImportError:
        return None

    y_index = _numeric(train_df[index_col]).astype("float32")
    y_tracking_error = _numeric(train_df[tracking_error_col]).astype("float32")
    if y_index.isna().any() or y_tracking_error.isna().any():
        return None

    index_model = clone(regressor)
    tracking_error_model = clone(regressor)
    index_model.fit(x_train, y_index)
    tracking_error_model.fit(x_train, y_tracking_error)
    return {
        "method": "tracking_index_plus_error",
        "index_return_regressor": index_model,
        "tracking_error_regressor": tracking_error_model,
        "index_return_target": index_col,
        "tracking_error_target": tracking_error_col,
        "regression_target": regression_target,
    }


def predict_return(regressor, x, decomposition: dict[str, Any] | None = None) -> dict[str, Any]:
    direct = np.asarray(regressor.predict(x), dtype="float64")
    if not decomposition:
        return {
            "method": "direct_fund_return",
            "prediction": direct,
            "direct_fund_return": direct,
            "index_return": None,
            "tracking_error": None,
        }

    index_model = decomposition.get("index_return_regressor")
    tracking_error_model = decomposition.get("tracking_error_regressor")
    if index_model is None or tracking_error_model is None:
        return {
            "method": "direct_fund_return",
            "prediction": direct,
            "direct_fund_return": direct,
            "index_return": None,
            "tracking_error": None,
        }

    index_return = np.asarray(index_model.predict(x), dtype="float64")
    tracking_error = np.asarray(tracking_error_model.predict(x), dtype="float64")
    return {
        "method": decomposition.get("method", "tracking_index_plus_error"),
        "prediction": index_return + tracking_error,
        "direct_fund_return": direct,
        "index_return": index_return,
        "tracking_error": tracking_error,
    }


def return_decomposition_metadata(decomposition: dict[str, Any] | None) -> dict[str, Any]:
    if not decomposition:
        return {"enabled": False, "method": "direct_fund_return"}
    return {
        "enabled": True,
        "method": decomposition.get("method", "tracking_index_plus_error"),
        "index_return_target": decomposition.get("index_return_target"),
        "tracking_error_target": decomposition.get("tracking_error_target"),
        "regression_target": decomposition.get("regression_target"),
    }


def _numeric(values):
    import pandas as pd

    return pd.to_numeric(values, errors="coerce")
