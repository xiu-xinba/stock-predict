from __future__ import annotations

import sys
import unittest
from pathlib import Path

try:
    import pandas as pd
except ImportError:
    pd = None

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.train_baseline import BaselineConfig, _labels_from_returns, _time_split


class TrainBaselineUnitTests(unittest.TestCase):
    def test_labels_from_returns_respects_flat_threshold(self) -> None:
        labels = _labels_from_returns([-0.2, 0.0, 0.2], flat_threshold_pct=0.05)

        self.assertEqual(labels.tolist(), [0, 1, 2])


@unittest.skipIf(pd is None, "pandas is not installed")
class TrainBaselinePandasTests(unittest.TestCase):
    def test_time_split_orders_by_existing_order(self) -> None:
        df = pd.DataFrame({
            "asof_time": pd.date_range("2026-01-01", periods=10, freq="D"),
            "value": range(10),
        })

        train_df, test_df = _time_split(df, 0.2)

        self.assertEqual(train_df["value"].tolist(), list(range(8)))
        self.assertEqual(test_df["value"].tolist(), [8, 9])


if __name__ == "__main__":
    unittest.main()
