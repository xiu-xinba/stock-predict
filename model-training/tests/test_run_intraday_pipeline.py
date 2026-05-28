from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.run_intraday_pipeline import load_intraday_pipeline_config


class RunIntradayPipelineTests(unittest.TestCase):
    def test_loads_intraday_pipeline_config(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            configs = root / "configs"
            configs.mkdir()
            config = configs / "intraday.yml"
            config.write_text(
                "\n".join([
                    "universe_path: configs/universe.csv",
                    "raw_dir: data/raw/intraday",
                    "samples_output_path: data/processed/intraday.csv",
                    "horizon_minutes: 3",
                    "purged_split: true",
                    "embargo_minutes: 3",
                    "candidates:",
                    "  - logistic_ridge",
                    "ablation_tests:",
                    "  - full_feature_model",
                    "promotion:",
                    "  registry_dir: model_registry",
                    "  min_balanced_accuracy: 0.31",
                    "  min_high_confidence_accuracy: 0.4",
                    "  min_score_delta: 0.002",
                    "  max_high_confidence_accuracy_drop: 0.01",
                    "  max_high_confidence_coverage_drop: 0.02",
                    "  mae_weight: 0.07",
                ]),
                encoding="utf-8",
            )

            loaded = load_intraday_pipeline_config(config)

        self.assertEqual(loaded.horizon_minutes, 3)
        self.assertEqual(loaded.min_balanced_accuracy, 0.31)
        self.assertEqual(loaded.min_high_confidence_accuracy, 0.4)
        self.assertEqual(loaded.min_score_delta, 0.002)
        self.assertEqual(loaded.max_high_confidence_accuracy_drop, 0.01)
        self.assertEqual(loaded.max_high_confidence_coverage_drop, 0.02)
        self.assertEqual(loaded.mae_weight, 0.07)
        self.assertEqual(loaded.candidates, ("logistic_ridge",))
        self.assertEqual(loaded.ablation_tests, ("full_feature_model",))
        self.assertEqual(loaded.embargo_minutes, 3)
        self.assertTrue(str(loaded.samples_output_path).endswith("data\\processed\\intraday.csv") or str(loaded.samples_output_path).endswith("data/processed/intraday.csv"))
