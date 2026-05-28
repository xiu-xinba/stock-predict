from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

import pandas as pd

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.prediction_logging import append_prediction_log
from fund_model_training.run_monitoring_cycle import MonitoringCycleConfig, run_monitoring_cycle


class RunMonitoringCycleTests(unittest.TestCase):
    def test_monitoring_cycle_backfills_evaluates_and_reports_drift(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            samples_path = root / "samples.csv"
            prediction_log_path = root / "prediction_log.csv"
            rows = []
            for i in range(80):
                asof_time = f"2026-05-{1 + i // 20:02d} 09:{30 + i % 20:02d}:00"
                rows.append({
                    "fund_code": "510300",
                    "asof_time": asof_time,
                    "future_return_pct_5m": 0.15 if i == 70 else 0.01,
                    "momentum_5d": 0.1 if i < 60 else 3.0,
                    "return_10d": 0.0,
                    "volatility_20d": 1.0,
                    "volume_ratio": 1.0,
                    "market_beta": 1.0,
                    "sector_momentum": 0.0,
                    "flow_signal": 0.0,
                    "mean_reversion": 0.0,
                })
            pd.DataFrame(rows).to_csv(samples_path, index=False)
            append_prediction_log({
                "fund_code": "510300",
                "asof_time": "2026-05-04T09:40:00",
                "created_at": "2026-05-04T09:40:02",
                "prediction": {
                    "horizon": "intraday_5m",
                    "predicted_change_pct": 0.12,
                    "direction": "up",
                    "direction_confidence": 0.75,
                    "is_actionable": True,
                },
                "model": {"candidate": "test_model", "model_path": "model.joblib"},
            }, prediction_log_path)

            summary = run_monitoring_cycle(MonitoringCycleConfig(
                prediction_log_path=prediction_log_path,
                samples_path=samples_path,
                feature_set="backend_v1",
                backfilled_log_output_path=root / "prediction_log_backfilled.csv",
                performance_report_output_path=root / "performance.json",
                drift_report_output_path=root / "drift.json",
                summary_output_path=root / "summary.json",
                reference_rows=40,
                current_rows=20,
                horizons=("intraday_5m",),
            ))

            saved = json.loads((root / "summary.json").read_text(encoding="utf-8"))

        self.assertFalse(summary["prediction_performance"]["skipped"])
        self.assertEqual(summary["prediction_performance"]["backfill"]["filled_rows"], 1)
        self.assertTrue(summary["drift"]["drift_detected"])
        self.assertEqual(saved["prediction_performance"]["performance"]["labeled_rows"], 1)

    def test_monitoring_cycle_skips_missing_prediction_log(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            samples_path = root / "samples.csv"
            pd.DataFrame([
                {
                    "fund_code": "510300",
                    "asof_time": f"2026-05-01 09:{30 + i:02d}:00",
                    "momentum_5d": float(i),
                    "return_10d": 0.0,
                    "volatility_20d": 1.0,
                    "volume_ratio": 1.0,
                    "market_beta": 1.0,
                    "sector_momentum": 0.0,
                    "flow_signal": 0.0,
                    "mean_reversion": 0.0,
                }
                for i in range(30)
            ]).to_csv(samples_path, index=False)

            summary = run_monitoring_cycle(MonitoringCycleConfig(
                prediction_log_path=root / "missing_prediction_log.csv",
                samples_path=samples_path,
                feature_set="backend_v1",
                backfilled_log_output_path=root / "prediction_log_backfilled.csv",
                performance_report_output_path=root / "performance.json",
                drift_report_output_path=root / "drift.json",
                summary_output_path=root / "summary.json",
                reference_rows=10,
                current_rows=10,
            ))

        self.assertTrue(summary["prediction_performance"]["skipped"])
        self.assertIn("drift_detected", summary["drift"])


if __name__ == "__main__":
    unittest.main()
