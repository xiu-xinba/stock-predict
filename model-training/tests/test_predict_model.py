from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.predict_model import (
    _bundle_calibration,
    _bundle_actionability_gate,
    _bundle_prediction_interval,
    _horizon_from_regression_target,
    _prediction_interval_bounds,
    _probability_payload,
    _reliability_from_calibration,
    _return_decomposition_payload,
    _safe_float,
    _signal_status,
)


class PredictModelUnitTests(unittest.TestCase):
    def test_probability_payload_uses_label_names(self) -> None:
        payload = _probability_payload([0.1, 0.2, 0.7])

        self.assertEqual(payload["down"], 0.1)
        self.assertEqual(payload["flat"], 0.2)
        self.assertEqual(payload["up"], 0.7)

    def test_probability_payload_uses_classifier_classes(self) -> None:
        payload = _probability_payload([0.25, 0.75], classes=[0, 2])

        self.assertEqual(payload["down"], 0.25)
        self.assertEqual(payload["flat"], 0.0)
        self.assertEqual(payload["up"], 0.75)

    def test_safe_float_handles_bad_values(self) -> None:
        self.assertEqual(_safe_float("1.2345678"), 1.234568)
        self.assertIsNone(_safe_float("bad"))

    def test_horizon_comes_from_regression_target(self) -> None:
        self.assertEqual(_horizon_from_regression_target("future_return_pct_5m"), ("intraday_5m", "未来5分钟"))
        self.assertEqual(_horizon_from_regression_target("future_return_pct_3m"), ("intraday_3m", "未来3分钟"))
        self.assertEqual(_horizon_from_regression_target("future_return_pct_next_day"), ("next_day", "下一个交易日"))

    def test_bundle_calibration_drives_reliability(self) -> None:
        calibration = _bundle_calibration({
            "champion_report": {
                "calibration": {
                    "ece": 0.04,
                    "mce": 0.10,
                    "brier_score": 0.22,
                },
            },
        })

        self.assertEqual(calibration["ece"], 0.04)
        self.assertEqual(_reliability_from_calibration(calibration), "model_calibrated")
        self.assertEqual(_reliability_from_calibration({"ece": 0.2}), "model_uncalibrated")

    def test_prediction_interval_comes_from_bundle_report(self) -> None:
        interval = {
            "enabled": True,
            "levels": {
                "0.90": {
                    "level": 0.9,
                    "lower_residual_quantile": -0.3,
                    "upper_residual_quantile": 0.7,
                }
            },
        }

        self.assertIs(_bundle_prediction_interval({"champion_report": {"prediction_interval": interval}}), interval)
        bounds = _prediction_interval_bounds(1.0, 0.8, {"prediction_interval": interval})

        self.assertEqual(bounds["low"], 0.7)
        self.assertEqual(bounds["high"], 1.7)
        self.assertEqual(bounds["method"], "empirical_residual_quantile")

    def test_return_decomposition_payload_exposes_design_formula(self) -> None:
        payload = _return_decomposition_payload(
            {
                "method": "tracking_index_plus_error",
                "index_return": [0.8],
                "tracking_error": [0.15],
                "direct_fund_return": [0.9],
            },
            {
                "method": "tracking_index_plus_error",
                "index_return_target": "future_index_return_pct_next_day",
                "tracking_error_target": "future_tracking_error_pct_next_day",
            },
        )

        self.assertTrue(payload["enabled"])
        self.assertEqual(payload["formula"], "fund_return = tracking_index_return + tracking_error")
        self.assertEqual(payload["index_return_pct"], 0.8)
        self.assertEqual(payload["tracking_error_pct"], 0.15)

    def test_signal_status_blocks_low_confidence_and_flat_actions(self) -> None:
        self.assertEqual(_signal_status("up", 0.59, 0.60), "low_confidence")
        self.assertEqual(_signal_status("flat", 0.80, 0.60), "no_signal")
        self.assertEqual(_signal_status("down", 0.80, 0.60), "actionable")

    def test_signal_status_blocks_models_below_actionability_gate(self) -> None:
        weak_gate = _bundle_actionability_gate({
            "champion_report": {
                "high_confidence": {"accuracy": 0.34, "coverage": 0.82},
                "calibration": {"ece": 0.45},
            },
        })
        strong_gate = _bundle_actionability_gate({
            "champion_report": {
                "high_confidence": {"accuracy": 0.62, "coverage": 0.40},
                "calibration": {"ece": 0.05},
            },
        })

        self.assertFalse(weak_gate["actionable"])
        self.assertEqual(_signal_status("up", 0.90, 0.60, weak_gate), "low_confidence")
        self.assertTrue(strong_gate["actionable"])
        self.assertEqual(_signal_status("up", 0.90, 0.60, strong_gate), "actionable")

    def test_actionability_gate_requires_enough_high_confidence_coverage(self) -> None:
        sparse_gate = _bundle_actionability_gate({
            "champion_report": {
                "high_confidence": {"accuracy": 0.75, "coverage": 0.01},
                "calibration": {"ece": 0.04},
            },
        })

        self.assertFalse(sparse_gate["actionable"])
        self.assertEqual(sparse_gate["reason"], "high_confidence_coverage_below_threshold")
        self.assertEqual(_signal_status("up", 0.90, 0.60, sparse_gate), "low_confidence")


if __name__ == "__main__":
    unittest.main()
