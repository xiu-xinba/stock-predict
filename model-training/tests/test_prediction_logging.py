from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.prediction_logging import (
    append_prediction_log,
    backfill_prediction_log,
    evaluate_prediction_log,
    prediction_payload_to_log_row,
)


class PredictionLoggingTests(unittest.TestCase):
    def test_prediction_payload_to_log_row_sets_due_time(self) -> None:
        row = prediction_payload_to_log_row(_payload("intraday_5m", "2026-05-22T14:55:00"))

        self.assertEqual(row["fund_code"], "510300")
        self.assertEqual(row["horizon"], "intraday_5m")
        self.assertEqual(row["label_due_time"], "2026-05-22 15:00:00")
        self.assertEqual(row["predicted_direction"], "up")
        self.assertEqual(row["signal_status"], "actionable")
        self.assertIn('"fear_score":0.4', row["feature_snapshot_json"])

    def test_append_and_backfill_prediction_log(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            log_path = root / "prediction_log.csv"
            samples_path = root / "samples.csv"
            output_path = root / "prediction_log_backfilled.csv"
            append_prediction_log(_payload("intraday_5m", "2026-05-22T14:55:00"), log_path, append=True)
            pd.DataFrame({
                "fund_code": ["510300"],
                "asof_time": ["2026-05-22 14:55:00"],
                "future_return_pct_5m": [0.08],
            }).to_csv(samples_path, index=False)

            result = backfill_prediction_log(log_path, samples_path, output_path, flat_threshold_pct=0.02)
            got = pd.read_csv(output_path, dtype={"fund_code": str})

        self.assertEqual(result["filled_rows"], 1)
        self.assertEqual(got.loc[0, "actual_direction"], "up")
        self.assertTrue(bool(got.loc[0, "hit_direction"]))

    def test_backfill_prediction_log_filters_horizon(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            log_path = root / "prediction_log.csv"
            samples_path = root / "samples.csv"
            output_path = root / "prediction_log_backfilled.csv"
            append_prediction_log(_payload("intraday_5m", "2026-05-22T14:55:00"), log_path, append=True)
            append_prediction_log(_payload("intraday_3m", "2026-05-22T14:56:00"), log_path, append=True)
            pd.DataFrame({
                "fund_code": ["510300"],
                "asof_time": ["2026-05-22 14:56:00"],
                "future_return_pct_3m": [0.08],
            }).to_csv(samples_path, index=False)

            result = backfill_prediction_log(
                log_path,
                samples_path,
                output_path,
                flat_threshold_pct=0.02,
                horizons=["intraday_3m"],
            )
            got = pd.read_csv(output_path, dtype={"fund_code": str})

        self.assertEqual(result["rows"], 1)
        self.assertEqual(result["filled_rows"], 1)
        self.assertEqual(got.loc[0, "horizon"], "intraday_3m")

    def test_evaluate_prediction_log(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            log_path = root / "prediction_log.csv"
            report_path = root / "prediction_report.json"
            pd.DataFrame({
                "prediction_id": ["a", "b"],
                "fund_code": ["510300", "510300"],
                "horizon": ["intraday_5m", "intraday_5m"],
                "asof_time": ["2026-05-22 14:55:00", "2026-05-22 14:56:00"],
                "created_at": ["2026-05-22 14:55:01", "2026-05-22 14:56:01"],
                "model_version": ["m", "m"],
                "feature_snapshot_id": ["f1", "f2"],
                "predicted_return": [0.05, -0.04],
                "predicted_direction": ["up", "down"],
                "confidence": [0.7, 0.4],
                "signal_status": ["actionable", "low_confidence"],
                "is_actionable": [True, False],
                "label_due_time": ["2026-05-22 15:00:00", "2026-05-22 15:01:00"],
                "actual_return": [0.08, 0.01],
                "actual_direction": ["up", "flat"],
            }).to_csv(log_path, index=False)

            report = evaluate_prediction_log(log_path, report_path, round_trip_cost_pct=0.01)

        self.assertEqual(report["labeled_rows"], 2)
        self.assertEqual(report["overall"]["direction_accuracy"], 0.5)
        self.assertEqual(report["overall"]["high_confidence_coverage"], 0.5)
        self.assertEqual(report["overall"]["signal_status"]["counts"]["low_confidence"], 1)
        self.assertEqual(report["overall"]["paper_trading"]["mean_cost_adjusted_return"], 0.025)


def _payload(horizon: str, asof_time: str) -> dict:
    return {
        "fund_code": "510300",
        "asof_time": asof_time,
        "created_at": "2026-05-22T14:55:01+08:00",
        "model": {
            "candidate": "logistic_ridge",
            "feature_set": "index_fund_intraday_v1",
            "model_path": "model_registry/intraday_5m/current.joblib",
        },
        "feature_snapshot": {
            "feature_set": "index_fund_intraday_v1",
            "features": {
                "fear_score": 0.4,
                "fund_return_1m": 0.02,
            },
        },
        "prediction": {
            "horizon": horizon,
            "direction": "up",
            "direction_confidence": 0.65,
            "predicted_change_pct": 0.05,
            "signal_status": "actionable",
            "is_actionable": True,
        },
    }


if __name__ == "__main__":
    unittest.main()
