from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.prediction_interval import empirical_prediction_interval_report, interval_bounds


class PredictionIntervalTests(unittest.TestCase):
    def test_empirical_interval_report_builds_coverage_blocks(self) -> None:
        report = empirical_prediction_interval_report(
            y_true=[1.0, 2.0, 3.0, 4.0],
            y_pred=[0.8, 2.2, 2.5, 4.4],
        )

        self.assertTrue(report["enabled"])
        self.assertEqual(report["method"], "empirical_residual_quantile")
        self.assertEqual(report["residual_count"], 4)
        self.assertIn("0.90", report["levels"])

    def test_interval_bounds_uses_residual_quantiles(self) -> None:
        report = {
            "enabled": True,
            "levels": {
                "0.90": {
                    "level": 0.9,
                    "lower_residual_quantile": -0.2,
                    "upper_residual_quantile": 0.5,
                    "empirical_coverage": 0.91,
                }
            },
        }

        bounds = interval_bounds(1.0, report, fallback_spread=2.0)

        self.assertEqual(bounds["low"], 0.8)
        self.assertEqual(bounds["high"], 1.5)
        self.assertEqual(bounds["method"], "empirical_residual_quantile")

    def test_interval_bounds_falls_back_for_old_bundles(self) -> None:
        bounds = interval_bounds(1.0, None, fallback_spread=0.3)

        self.assertEqual(bounds["low"], 0.7)
        self.assertEqual(bounds["high"], 1.3)
        self.assertEqual(bounds["method"], "heuristic_spread")


if __name__ == "__main__":
    unittest.main()
