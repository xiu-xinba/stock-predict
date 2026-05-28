from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.build_intraday_index_fund_dataset import _append_history, build_intraday_index_fund_dataset


class BuildIntradayIndexFundDatasetTests(unittest.TestCase):
    def test_builds_batch_intraday_samples_with_mock_collectors(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            universe = root / "universe.csv"
            universe.write_text(
                "\n".join([
                    "fund_code,fund_name,tracking_index,market",
                    "510300,沪深300ETF,sh000300,CN",
                    "159915,创业板ETF,sz399006,CN",
                ]),
                encoding="utf-8",
            )

            def fake_fund(symbol: str, **kwargs):
                times = pd.date_range("2026-05-22 09:30:00", periods=12, freq="min")
                offset = 0.2 if symbol == "159915" else 0.0
                return pd.DataFrame({
                    "fund_code": [symbol] * len(times),
                    "timestamp": times,
                    "available_time": times,
                    "price": [4.0 + offset + i * 0.01 for i in range(len(times))],
                    "iopv": [0.0] * len(times),
                    "premium_pct": [0.0] * len(times),
                    "volume": [100 + i for i in range(len(times))],
                    "amount": [1000 + i for i in range(len(times))],
                    "bid_ask_spread": [0.001] * len(times),
                })

            def fake_index(symbol: str, **kwargs):
                times = pd.date_range("2026-05-22 09:30:00", periods=12, freq="min")
                return pd.DataFrame({
                    "index_code": [symbol] * len(times),
                    "timestamp": times,
                    "available_time": times,
                    "price": [3900 + i for i in range(len(times))],
                    "return": [0.0] * len(times),
                    "volume": [1000 + i for i in range(len(times))],
                    "amount": [1_000_000 + i for i in range(len(times))],
                })

            with patch("fund_model_training.build_intraday_index_fund_dataset.collect_etf_intraday", side_effect=fake_fund), \
                    patch("fund_model_training.build_intraday_index_fund_dataset.collect_index_intraday", side_effect=fake_index):
                summary = build_intraday_index_fund_dataset(
                    universe_path=universe,
                    raw_dir=root / "raw",
                    history_dir=root / "history",
                    output_path=root / "samples.csv",
                    horizon_minutes=5,
                )

            samples = pd.read_csv(root / "samples.csv", dtype={"fund_code": str})
            history_exists = (root / "history" / "fund_intraday_history.csv").exists()

        self.assertTrue(summary["ok"])
        self.assertEqual(summary["funds_collected"], 2)
        self.assertEqual(summary["indexes_collected"], 2)
        self.assertGreater(summary["sample_rows"], 0)
        self.assertEqual(set(samples["fund_code"]), {"510300", "159915"})
        self.assertTrue(history_exists)

    def test_append_history_deduplicates_keys(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            history = root / "history.csv"
            first = pd.DataFrame({
                "fund_code": ["510300", "510300"],
                "timestamp": ["2026-05-22 09:30:00", "2026-05-22 09:31:00"],
                "price": [4.0, 4.1],
            })
            second = pd.DataFrame({
                "fund_code": ["510300", "510300"],
                "timestamp": ["2026-05-22 09:31:00", "2026-05-22 09:32:00"],
                "price": [4.2, 4.3],
            })

            _append_history(first, history, ["fund_code", "timestamp"], dtype={"fund_code": str})
            _append_history(second, history, ["fund_code", "timestamp"], dtype={"fund_code": str})
            got = pd.read_csv(history, dtype={"fund_code": str})

        self.assertEqual(len(got), 3)
        self.assertEqual(float(got.loc[got["timestamp"] == "2026-05-22 09:31:00", "price"].iloc[0]), 4.2)


if __name__ == "__main__":
    unittest.main()
