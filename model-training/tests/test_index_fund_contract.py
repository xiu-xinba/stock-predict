from __future__ import annotations

import sys
import unittest
from pathlib import Path

try:
    import pandas as pd
except ImportError:
    pd = None

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.index_fund_contract import (
    data_dictionary,
    get_table_spec,
    known_tables,
    validate_columns,
)

if pd is not None:
    from fund_model_training.index_fund_contract import validate_frame


class IndexFundContractTests(unittest.TestCase):
    def test_contract_contains_first_phase_tables(self) -> None:
        tables = set(known_tables())

        self.assertIn("fund_daily", tables)
        self.assertIn("futures_bar", tables)
        self.assertIn("panic_factor", tables)
        self.assertIn("prediction_log", tables)

    def test_required_columns_are_reported(self) -> None:
        missing = validate_columns("fund_daily", ["fund_code", "trade_date", "nav"])

        self.assertIn("available_time", missing)
        self.assertIn("adjusted_nav", missing)

    def test_data_dictionary_is_json_friendly(self) -> None:
        dictionary = data_dictionary()
        spec = get_table_spec("panic_factor")

        self.assertEqual(dictionary["panic_factor"]["name"], spec.name)
        self.assertIn("fear_score", dictionary["panic_factor"]["required_columns"])


@unittest.skipIf(pd is None, "pandas is not installed")
class IndexFundFrameValidationTests(unittest.TestCase):

    def test_invalid_datetime_is_reported(self) -> None:
        df = pd.DataFrame({
            "fund_code": ["510300"],
            "trade_date": ["not-a-date"],
            "available_time": ["2026-05-23 16:00:00"],
            "nav": [1.23],
            "adjusted_nav": [1.23],
        })

        errors = validate_frame("fund_daily", df)

        self.assertTrue(any("trade_date" in error for error in errors))

    def test_available_time_cannot_precede_event_time(self) -> None:
        df = pd.DataFrame({
            "contract": ["IF2606"],
            "underlying": ["IF"],
            "timestamp": ["2026-05-23 10:00:00"],
            "available_time": ["2026-05-23 09:59:59"],
            "price": [4123.5],
        })

        errors = validate_frame("futures_bar", df)

        self.assertTrue(any("available_time" in error for error in errors))

    def test_empty_optional_datetime_is_allowed(self) -> None:
        df = pd.DataFrame({
            "fund_code": ["510300"],
            "fund_name": ["沪深300ETF"],
            "fund_type": ["ETF"],
            "tracking_index": ["CSI300"],
            "market": ["CN"],
            "is_etf": [True],
            "inception_date": [None],
        })

        errors = validate_frame("dim_fund", df)

        self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
