from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.build_intraday_samples import build_intraday_samples
from fund_model_training.schema import get_feature_names


class BuildIntradaySamplesTests(unittest.TestCase):
    def test_builds_five_minute_labels_and_features(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            times = pd.date_range("2026-05-22 09:30:00", periods=12, freq="min")
            fund = pd.DataFrame({
                "fund_code": ["510300"] * len(times),
                "timestamp": times,
                "available_time": times,
                "price": [4.00, 4.01, 4.02, 4.03, 4.04, 4.10, 4.12, 4.11, 4.13, 4.15, 4.14, 4.16],
                "premium_pct": [0.01] * len(times),
                "volume": [100 + i * 3 for i in range(len(times))],
                "bid_ask_spread": [0.001] * len(times),
            })
            index = pd.DataFrame({
                "index_code": ["sh000300"] * len(times),
                "timestamp": times,
                "available_time": times,
                "price": [3900 + i * 2 for i in range(len(times))],
                "return": [0.0] * len(times),
                "volume": [1000 + i * 10 for i in range(len(times))],
                "amount": [1_000_000 + i * 1000 for i in range(len(times))],
            })
            panic = pd.DataFrame({
                "market": ["CN"],
                "timestamp": [times[0]],
                "fear_score": [0.42],
                "iv_component": [0.3],
                "flow_component": [0.4],
                "news_component": [0.2],
                "limit_component": [0.1],
            })
            fund_path = root / "fund_intraday.csv"
            index_path = root / "index_intraday.csv"
            panic_path = root / "panic.csv"
            fund.to_csv(fund_path, index=False)
            index.to_csv(index_path, index=False)
            panic.to_csv(panic_path, index=False)

            samples = build_intraday_samples(
                fund_intraday_path=fund_path,
                index_intraday_path=index_path,
                panic_factor_path=panic_path,
                fund_code="510300",
                tracking_index="sh000300",
                horizon_minutes=5,
            )

        self.assertGreater(len(samples), 0)
        self.assertIn("future_return_pct_5m", samples.columns)
        self.assertIn("future_index_return_pct_5m", samples.columns)
        self.assertIn("future_tracking_error_pct_5m", samples.columns)
        self.assertIn("fund_index_spread_5m", samples.columns)
        self.assertTrue(set(get_feature_names("index_fund_intraday_v1")).issubset(samples.columns))
        first_return = samples.loc[0, "future_return_pct_5m"]
        self.assertAlmostEqual(first_return, 2.5, places=4)
        self.assertAlmostEqual(float(samples.loc[0, "future_index_return_pct_5m"]), 0.2564, places=4)
        self.assertAlmostEqual(
            float(samples.loc[0, "future_tracking_error_pct_5m"]),
            first_return - float(samples.loc[0, "future_index_return_pct_5m"]),
            places=4,
        )
        self.assertEqual(int(samples.loc[0, "label"]), 2)
        self.assertEqual(float(samples.loc[0, "fear_score"]), 0.42)


if __name__ == "__main__":
    unittest.main()
