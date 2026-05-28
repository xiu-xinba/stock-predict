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
    from fund_model_training.build_panic_proxy import build_panic_proxy_components


@unittest.skipIf(pd is None, "pandas is not installed")
class PanicProxyTests(unittest.TestCase):
    def test_builds_proxy_components_from_index_and_futures(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            dates = pd.date_range("2026-01-01", periods=6, freq="D")
            pd.DataFrame({
                "index_code": ["sh000300"] * len(dates),
                "trade_date": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "open": range(100, 106),
                "high": range(101, 107),
                "low": range(99, 105),
                "close": [100, 98, 99, 97, 96, 101],
            }).to_csv(root / "index.csv", index=False)
            pd.DataFrame({
                "contract": ["IF0"] * len(dates),
                "underlying": ["IF"] * len(dates),
                "timestamp": dates,
                "available_time": [d.strftime("%Y-%m-%d 16:00:00") for d in dates],
                "price": [100, 99, 101, 98, 97, 102],
                "open_interest": [10, 12, 13, 15, 14, 16],
            }).to_csv(root / "futures.csv", index=False)

            components = build_panic_proxy_components(root / "index.csv", root / "futures.csv")

        self.assertEqual(len(components), len(dates))
        self.assertIn("iv_component", components.columns)
        self.assertIn("news_component", components.columns)
        self.assertTrue((components["market"] == "CN").all())


if __name__ == "__main__":
    unittest.main()
