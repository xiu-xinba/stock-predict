from __future__ import annotations

import pandas as pd

from .schema import LABEL_TO_ID


def labels_from_future_return(values: pd.Series, flat_threshold_pct: float) -> pd.Series:
    numeric = pd.to_numeric(values, errors="coerce")
    labels = pd.Series(pd.NA, index=values.index, dtype="Int64")
    labels.loc[numeric < -flat_threshold_pct] = LABEL_TO_ID["down"]
    labels.loc[numeric > flat_threshold_pct] = LABEL_TO_ID["up"]
    flat_mask = numeric.notna() & numeric.between(
        -flat_threshold_pct,
        flat_threshold_pct,
        inclusive="both",
    )
    labels.loc[flat_mask] = LABEL_TO_ID["flat"]
    return labels


def ensure_label(
    df: pd.DataFrame,
    label_column: str,
    future_return_column: str,
    flat_threshold_pct: float,
) -> pd.DataFrame:
    out = df.copy()
    if label_column in out.columns:
        raw = out[label_column]
        if pd.api.types.is_numeric_dtype(raw):
            out[label_column] = pd.to_numeric(raw, errors="coerce")
        else:
            out[label_column] = raw.astype(str).str.lower().map(LABEL_TO_ID)
    elif future_return_column in out.columns:
        out[label_column] = labels_from_future_return(out[future_return_column], flat_threshold_pct)
    else:
        raise ValueError(
            f"Missing label data. Provide '{label_column}' or '{future_return_column}'."
        )

    out = out.dropna(subset=[label_column])
    out[label_column] = out[label_column].astype("int64")
    invalid = sorted(set(out[label_column].unique()) - set(LABEL_TO_ID.values()))
    if invalid:
        raise ValueError(f"Invalid label ids: {invalid}. Expected 0=down, 1=flat, 2=up.")
    return out
