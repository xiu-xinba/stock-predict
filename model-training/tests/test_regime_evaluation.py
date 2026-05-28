from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.regime_evaluation import market_regime_report


class RegimeEvaluationTests(unittest.TestCase):
    def test_market_regime_report_builds_design_slices(self) -> None:
        test_df = pd.DataFrame({
            "index_return_5d": [-3.0, -2.0, -0.1, 0.2, 1.5, 3.0],
            "fear_score": [0.1, 0.2, 0.3, 0.8, 0.9, 1.0],
            "index_volatility_20d": [0.5, 0.6, 0.7, 1.5, 1.8, 2.0],
        })

        report = market_regime_report(
            test_df=test_df,
            y_true_cls=[0, 0, 1, 1, 2, 2],
            y_pred_cls=[0, 1, 1, 0, 0, 0],
            y_true_reg=[-1, -0.5, 0, 0.1, 0.8, 1.2],
            y_pred_reg=[-0.8, 0.2, 0.1, 0.0, 0.7, -0.5],
            probabilities=[
                [0.8, 0.1, 0.1],
                [0.2, 0.6, 0.2],
                [0.1, 0.7, 0.2],
                [0.2, 0.7, 0.1],
                [0.1, 0.2, 0.7],
                [0.6, 0.2, 0.2],
            ],
            high_confidence_threshold=0.6,
            min_slice_rows=2,
        )

        self.assertEqual(report["groups"]["market_trend"]["status"], "ok")
        self.assertIn("bear", report["groups"]["market_trend"]["slices"])
        self.assertIn("high_panic", report["groups"]["panic"]["slices"])
        self.assertIn("high_volatility", report["groups"]["volatility"]["slices"])
        self.assertGreaterEqual(len(report["weak_slices"]), 1)


if __name__ == "__main__":
    unittest.main()
