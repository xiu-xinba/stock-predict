from __future__ import annotations

import sys
import unittest
from pathlib import Path

try:
    import pandas as pd
except ImportError:
    pd = None

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.collect_data import DATASET_TABLES
from fund_model_training.collectors.common import infer_market_from_name, standardize_code

if pd is not None:
    from fund_model_training.collectors.akshare_adapter import _market_close_time, _sina_exchange_symbol


class CollectDataTests(unittest.TestCase):
    def test_dataset_routes_to_contract_tables(self) -> None:
        self.assertEqual(DATASET_TABLES["etf_universe"], "dim_fund")
        self.assertEqual(DATASET_TABLES["etf_spot"], "fund_intraday")
        self.assertEqual(DATASET_TABLES["futures_main"], "futures_bar")

    def test_fund_code_standardization(self) -> None:
        self.assertEqual(standardize_code("510300"), "510300")
        self.assertEqual(standardize_code("159915.0"), "159915")
        self.assertEqual(standardize_code("51050"), "051050")

    def test_market_inference_from_fund_name(self) -> None:
        self.assertEqual(infer_market_from_name("恒生科技ETF"), "HK")
        self.assertEqual(infer_market_from_name("纳指100ETF"), "GLOBAL")
        self.assertEqual(infer_market_from_name("沪深300ETF"), "CN")


@unittest.skipIf(pd is None, "pandas is not installed")
class CollectDataPandasTests(unittest.TestCase):
    def test_daily_available_time_defaults_to_market_close(self) -> None:
        values = pd.Series(["2026-05-22 00:00:00"])

        self.assertEqual(_market_close_time(values).iloc[0], "2026-05-22 16:00:00")

    def test_sina_exchange_symbol_inference(self) -> None:
        self.assertEqual(_sina_exchange_symbol("510300"), "sh510300")
        self.assertEqual(_sina_exchange_symbol("159915"), "sz159915")


if __name__ == "__main__":
    unittest.main()
