from __future__ import annotations

from datetime import timedelta
from typing import Any

import pandas as pd


def time_series_split(df: pd.DataFrame, test_size: float) -> tuple[pd.DataFrame, pd.DataFrame]:
    if not 0 < test_size < 1:
        raise ValueError("test_size must be between 0 and 1.")
    ordered = df.sort_values("asof_time").reset_index(drop=True)
    split_idx = int(len(ordered) * (1.0 - test_size))
    if split_idx <= 0 or split_idx >= len(ordered):
        raise ValueError("Not enough rows for a train/test split.")
    return ordered.iloc[:split_idx].copy(), ordered.iloc[split_idx:].copy()


def purged_time_series_split(
    df: pd.DataFrame,
    test_size: float,
    label_horizon: timedelta,
    embargo: timedelta = timedelta(0),
) -> tuple[pd.DataFrame, pd.DataFrame, dict[str, Any]]:
    train, test = time_series_split(df, test_size)
    train["asof_time"] = pd.to_datetime(train["asof_time"], errors="coerce")
    test["asof_time"] = pd.to_datetime(test["asof_time"], errors="coerce")
    train = train.dropna(subset=["asof_time"]).copy()
    test = test.dropna(subset=["asof_time"]).copy()
    if train.empty or test.empty:
        raise ValueError("Not enough valid asof_time rows for a purged split.")

    original_train_rows = len(train)
    test_start = test["asof_time"].min()
    cutoff = test_start - embargo
    keep = (train["asof_time"] + label_horizon) < cutoff
    train = train.loc[keep].copy()
    if train.empty:
        raise ValueError("Purged split removed every training row; reduce test_size or embargo.")

    metadata = {
        "type": "purged_time_holdout",
        "label_horizon_seconds": int(label_horizon.total_seconds()),
        "embargo_seconds": int(embargo.total_seconds()),
        "original_train_rows": int(original_train_rows),
        "purged_train_rows": int(len(train)),
        "removed_train_rows": int(original_train_rows - len(train)),
        "test_start": str(test_start),
        "purge_cutoff": str(cutoff),
    }
    return train, test, metadata
