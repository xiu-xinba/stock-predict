from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

import numpy as np
import pandas as pd

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.drift_report import build_drift_report, ks_statistic, population_stability_index


class DriftReportTests(unittest.TestCase):
    def test_distribution_metrics_detect_shift(self) -> None:
        reference = np.zeros(30)
        current = np.ones(30) * 5

        self.assertGreaterEqual(ks_statistic(reference, current), 0.9)
        self.assertGreater(population_stability_index(reference, current), 1.0)

    def test_build_drift_report_writes_json(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            samples_path = root / "samples.csv"
            output_path = root / "drift.json"
            rows = []
            for i in range(80):
                rows.append({
                    "fund_code": "510300",
                    "asof_time": f"2026-05-{1 + i // 20:02d} 09:{30 + i % 20:02d}:00",
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

            report = build_drift_report(
                samples_path=samples_path,
                feature_set="backend_v1",
                output_path=output_path,
                reference_rows=40,
                current_rows=20,
                psi_threshold=0.2,
                ks_threshold=0.2,
            )

            saved = json.loads(output_path.read_text(encoding="utf-8"))

        self.assertTrue(report["drift_detected"])
        self.assertEqual(saved["feature_set"], "backend_v1")
        drifted = {item["feature"] for item in saved["features"] if item["drifted"]}
        self.assertIn("momentum_5d", drifted)


if __name__ == "__main__":
    unittest.main()
