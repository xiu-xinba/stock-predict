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
    from fund_model_training.build_index_fund_dataset import _load_universe, _safe_name


class BuildIndexFundDatasetUnitTests(unittest.TestCase):
    def test_safe_name_removes_non_alnum(self) -> None:
        self.assertEqual(_safe_name("sh.000300"), "sh_000300")


@unittest.skipIf(pd is None, "pandas is not installed")
class BuildIndexFundDatasetPandasTests(unittest.TestCase):
    def test_load_universe_defaults_optional_columns(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "universe.csv"
            pd.DataFrame({
                "fund_code": ["510300"],
                "tracking_index": ["sh000300"],
                "market": ["CN"],
            }).to_csv(path, index=False)

            universe = _load_universe(path)

        self.assertEqual(universe.loc[0, "fund_code"], "510300")
        self.assertEqual(universe.loc[0, "fund_name"], "510300")
        self.assertIn("futures_symbol", universe.columns)


if __name__ == "__main__":
    unittest.main()
