from __future__ import annotations

import pandas as pd


def time_series_split(df: pd.DataFrame, test_size: float) -> tuple[pd.DataFrame, pd.DataFrame]:
    if not 0 < test_size < 1:
        raise ValueError("test_size must be between 0 and 1.")
    ordered = df.sort_values("asof_time").reset_index(drop=True)
    split_idx = int(len(ordered) * (1.0 - test_size))
    if split_idx <= 0 or split_idx >= len(ordered):
        raise ValueError("Not enough rows for a train/test split.")
    return ordered.iloc[:split_idx].copy(), ordered.iloc[split_idx:].copy()
