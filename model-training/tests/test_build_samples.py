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
    from fund_model_training.build_samples import build_daily_weekly_samples
    from fund_model_training.features import prepare_features


@unittest.skipIf(pd is None, "pandas is not installed")
class BuildSamplesTests(unittest.TestCase):
    def test_build_daily_weekly_samples_with_market_features(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            dates = pd.date_range("2026-01-01", periods=8, freq="D")
            pd.DataFrame({
                "fund_code": ["510300"] * len(dates),
                "trade_date": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "nav": [1.00, 1.01, 1.03, 1.02, 1.04, 1.05, 1.06, 1.08],
                "adjusted_nav": [1.00, 1.01, 1.03, 1.02, 1.04, 1.05, 1.06, 1.08],
            }).to_csv(root / "fund_daily.csv", index=False)
            pd.DataFrame({
                "fund_code": ["510300"],
                "fund_name": ["沪深300ETF"],
                "fund_type": ["ETF"],
                "tracking_index": ["sh000300"],
                "market": ["CN"],
                "is_etf": [True],
            }).to_csv(root / "dim_fund.csv", index=False)
            pd.DataFrame({
                "index_code": ["sh000300"] * len(dates),
                "trade_date": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "open": range(100, 108),
                "high": range(101, 109),
                "low": range(99, 107),
                "close": [100, 101, 102, 101, 103, 104, 105, 107],
            }).to_csv(root / "index_daily.csv", index=False)
            pd.DataFrame({
                "contract": ["IF0"] * len(dates),
                "underlying": ["IF"] * len(dates),
                "timestamp": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "price": [100, 102, 101, 103, 104, 105, 103, 106],
                "open_interest": [10, 11, 12, 12, 13, 14, 14, 15],
            }).to_csv(root / "futures.csv", index=False)
            pd.DataFrame({
                "market": ["CN"] * len(dates),
                "timestamp": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "fear_score": [0.1, 0.2, 0.3, 0.2, 0.4, 0.3, 0.2, 0.1],
                "iv_component": [0.1] * len(dates),
                "flow_component": [0.2] * len(dates),
                "news_component": [0.3] * len(dates),
                "limit_component": [0.4] * len(dates),
            }).to_csv(root / "panic.csv", index=False)

            samples = build_daily_weekly_samples(
                fund_daily_path=root / "fund_daily.csv",
                dim_fund_path=root / "dim_fund.csv",
                index_daily_path=root / "index_daily.csv",
                futures_path=root / "futures.csv",
                panic_factor_path=root / "panic.csv",
            )

        self.assertGreater(len(samples), 0)
        self.assertIn("future_return_pct_next_day", samples.columns)
        self.assertIn("fund_tracking_error_1d", samples.columns)
        self.assertIn("futures_return_1d", samples.columns)
        self.assertIn("fear_score", samples.columns)

        prepared, features = prepare_features(samples, "index_fund_daily_v1")
        self.assertIn("index_return_1d", features)
        self.assertFalse(prepared[features].isna().any().any())


if __name__ == "__main__":
    unittest.main()
