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
    from fund_model_training.collectors.akshare_adapter import (
        _candidate_index_market_ids,
        _eastmoney_trends_to_frame,
        _market_close_time,
        _mootdx_bars_to_frame,
        _mootdx_frequency,
        _mootdx_index_mapping,
        _normalize_index_symbol,
        _sina_exchange_symbol,
        _tencent_fund_symbol,
        _tencent_index_symbol,
        _tencent_minute_to_frame,
    )


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

    def test_index_symbol_normalization_for_eastmoney_fallback(self) -> None:
        self.assertEqual(_normalize_index_symbol("sh000300"), "000300")
        self.assertEqual(_normalize_index_symbol("399006"), "399006")
        self.assertEqual(_candidate_index_market_ids("sz399006")[0], 0)

    def test_tencent_symbol_inference(self) -> None:
        self.assertEqual(_tencent_fund_symbol("510300"), "sh510300")
        self.assertEqual(_tencent_fund_symbol("159915"), "sz159915")
        self.assertEqual(_tencent_index_symbol("sh000300"), "sh000300")
        self.assertEqual(_tencent_index_symbol("399006"), "sz399006")

    def test_mootdx_helpers(self) -> None:
        self.assertEqual(_mootdx_frequency("1"), "1m")
        self.assertEqual(_mootdx_frequency("5m"), "5m")
        self.assertEqual(_mootdx_index_mapping("sh000300"), (62, "000300"))
        self.assertIsNone(_mootdx_index_mapping("sz399006"))

    def test_mootdx_bars_payload_to_frame(self) -> None:
        raw = pd.DataFrame({
            "datetime": ["2026-05-22 09:30", "2026-05-22 09:31"],
            "open": [4.0, 4.1],
            "close": [4.1, 4.2],
            "high": [4.2, 4.3],
            "low": [3.9, 4.0],
            "volume": [1000, 1100],
            "amount": [4000, 4510],
        })

        frame = _mootdx_bars_to_frame(raw)

        self.assertEqual(list(frame.columns), ["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"])
        self.assertEqual(len(frame), 2)
        self.assertEqual(str(frame.loc[0, "时间"]), "2026-05-22 09:30:00")
        self.assertEqual(float(frame.loc[1, "收盘"]), 4.2)

    def test_eastmoney_trends_payload_to_frame(self) -> None:
        payload = {
            "data": {
                "trends": [
                    "2026-05-22 09:30,4.00,4.01,4.02,3.99,100,400000,4.005",
                    "2026-05-22 09:31,4.01,4.03,4.04,4.01,110,440000,4.020",
                ]
            }
        }

        frame = _eastmoney_trends_to_frame(payload)

        self.assertEqual(list(frame.columns), ["时间", "开盘", "收盘", "最高", "最低", "成交量", "成交额", "均价"])
        self.assertEqual(len(frame), 2)
        self.assertEqual(float(frame.loc[0, "收盘"]), 4.01)

    def test_tencent_minute_payload_to_frame(self) -> None:
        payload = {
            "data": {
                "sh510300": {
                    "data": {
                        "date": "20260522",
                        "data": [
                            "0930 4.828 27095 13081466.00",
                            "0931 4.838 159115 76891020.00",
                        ],
                    }
                }
            }
        }

        frame = _tencent_minute_to_frame(payload, "sh510300")

        self.assertEqual(len(frame), 2)
        self.assertEqual(str(frame.loc[0, "时间"]), "2026-05-22 09:30:00")
        self.assertEqual(float(frame.loc[1, "收盘"]), 4.838)


if __name__ == "__main__":
    unittest.main()
