from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

try:
    import pandas as pd
except ImportError:
    pd = None

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

if pd is not None:
    from fund_model_training.build_panic_factor import DEFAULT_WEIGHTS, build_panic_factor


@unittest.skipIf(pd is None, "pandas is not installed")
class PanicFactorTests(unittest.TestCase):
    def test_builds_point_in_time_panic_factor(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            input_path = Path(tmp) / "components.csv"
            pd.DataFrame({
                "market": ["CN", "CN", "CN"],
                "timestamp": ["2026-01-01", "2026-01-02", "2026-01-03"],
                "available_time": ["2026-01-01 16:00:00", "2026-01-02 16:00:00", "2026-01-03 16:00:00"],
                "iv_component": [10, 20, 15],
                "flow_component": [0.1, 0.2, 0.3],
                "news_component": [1, 3, 2],
                "limit_component": [5, 4, 6],
            }).to_csv(input_path, index=False)

            out = build_panic_factor(
                input_path=input_path,
                market=None,
                timestamp_col="timestamp",
                available_time_col="available_time",
                weights=DEFAULT_WEIGHTS,
            )

        self.assertEqual(out["market"].tolist(), ["CN", "CN", "CN"])
        self.assertTrue(out["fear_score"].between(0, 1).all())
        self.assertIn("iv_component", out.columns)


if __name__ == "__main__":
    unittest.main()
