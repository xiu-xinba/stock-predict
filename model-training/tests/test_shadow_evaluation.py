from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.shadow_evaluation import ShadowEvaluationConfig, evaluate_shadow_model


class ShadowEvaluationTests(unittest.TestCase):
    def test_shadow_evaluation_passes_when_challenger_beats_gates(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            log_path = root / "prediction_log_backfilled.csv"
            output_path = root / "shadow_report.json"
            pd.DataFrame({
                "prediction_id": ["c1", "c2", "m1", "m2"],
                "fund_code": ["510300"] * 4,
                "horizon": ["next_day"] * 4,
                "asof_time": ["2026-05-20", "2026-05-21", "2026-05-20", "2026-05-21"],
                "created_at": ["2026-05-20"] * 4,
                "model_version": ["challenger"] * 2 + ["champion"] * 2,
                "feature_snapshot_id": ["f1", "f2", "f3", "f4"],
                "predicted_return": [0.1, -0.1, -0.1, -0.1],
                "predicted_direction": ["up", "down", "down", "down"],
                "confidence": [0.7, 0.7, 0.7, 0.7],
                "is_actionable": [True, True, True, True],
                "label_due_time": ["2026-05-21", "2026-05-22", "2026-05-21", "2026-05-22"],
                "actual_return": [0.2, -0.2, 0.2, -0.2],
                "actual_direction": ["up", "down", "up", "down"],
            }).to_csv(log_path, index=False)

            report = evaluate_shadow_model(ShadowEvaluationConfig(
                prediction_log_path=log_path,
                output_path=output_path,
                challenger_model_version="challenger",
                champion_model_version="champion",
                min_shadow_days=2,
                min_labeled_rows=2,
                min_direction_accuracy=0.5,
                min_cost_adjusted_return=0.0,
                round_trip_cost_pct=0.01,
            ))

            self.assertTrue(report["passed"])
            self.assertEqual(report["challenger"]["shadow_days"], 2)
            self.assertTrue(output_path.exists())

    def test_shadow_evaluation_rejects_short_shadow_run(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            log_path = root / "prediction_log_backfilled.csv"
            output_path = root / "shadow_report.json"
            pd.DataFrame({
                "prediction_id": ["c1"],
                "fund_code": ["510300"],
                "horizon": ["next_day"],
                "asof_time": ["2026-05-20"],
                "created_at": ["2026-05-20"],
                "model_version": ["challenger"],
                "feature_snapshot_id": ["f1"],
                "predicted_return": [0.1],
                "predicted_direction": ["up"],
                "confidence": [0.7],
                "is_actionable": [True],
                "label_due_time": ["2026-05-21"],
                "actual_return": [0.2],
                "actual_direction": ["up"],
            }).to_csv(log_path, index=False)

            report = evaluate_shadow_model(ShadowEvaluationConfig(
                prediction_log_path=log_path,
                output_path=output_path,
                challenger_model_version="challenger",
                min_shadow_days=2,
                min_labeled_rows=1,
            ))

            self.assertFalse(report["passed"])
            self.assertTrue(any("shadow_days below threshold" in reason for reason in report["reasons"]))

    def test_shadow_evaluation_can_skip_missing_log(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            output_path = root / "shadow_report.json"

            report = evaluate_shadow_model(ShadowEvaluationConfig(
                prediction_log_path=root / "missing.csv",
                output_path=output_path,
                challenger_model_version="challenger",
                skip_missing_prediction_log=True,
            ))

            self.assertTrue(report["skipped"])
            self.assertFalse(report["passed"])
            self.assertTrue(output_path.exists())


if __name__ == "__main__":
    unittest.main()
