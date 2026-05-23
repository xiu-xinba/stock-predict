from __future__ import annotations

import sys
import unittest
from pathlib import Path

try:
    import pandas as pd
except ImportError:
    pd = None

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

if pd is not None:
    from fund_model_training.labels import ensure_label, labels_from_future_return
    from fund_model_training.schema import LABEL_TO_ID


@unittest.skipIf(pd is None, "pandas is not installed")
class LabelTests(unittest.TestCase):
    def test_missing_future_return_is_not_marked_flat(self) -> None:
        labels = labels_from_future_return(
            pd.Series([-0.2, 0.0, 0.2, None, "bad"]),
            flat_threshold_pct=0.05,
        )

        self.assertEqual(labels.iloc[0], LABEL_TO_ID["down"])
        self.assertEqual(labels.iloc[1], LABEL_TO_ID["flat"])
        self.assertEqual(labels.iloc[2], LABEL_TO_ID["up"])
        self.assertTrue(pd.isna(labels.iloc[3]))
        self.assertTrue(pd.isna(labels.iloc[4]))

    def test_ensure_label_drops_missing_future_return_rows(self) -> None:
        df = pd.DataFrame({
            "fund_code": ["000001", "000002", "000003"],
            "asof_time": ["2026-01-01", "2026-01-01", "2026-01-01"],
            "future_return_pct_next_day": [0.2, None, -0.2],
        })

        labeled = ensure_label(
            df,
            label_column="label",
            future_return_column="future_return_pct_next_day",
            flat_threshold_pct=0.05,
        )

        self.assertEqual(labeled["fund_code"].tolist(), ["000001", "000003"])
        self.assertEqual(labeled["label"].tolist(), [LABEL_TO_ID["up"], LABEL_TO_ID["down"]])


if __name__ == "__main__":
    unittest.main()
