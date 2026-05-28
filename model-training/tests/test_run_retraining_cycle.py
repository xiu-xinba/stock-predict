from __future__ import annotations

import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.run_retraining_cycle import load_retraining_cycle_config


class RunRetrainingCycleTests(unittest.TestCase):
    def test_loads_cycle_config_with_resolved_paths(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            configs = root / "configs"
            configs.mkdir()
            config = configs / "cycle.yml"
            config.write_text(
                "\n".join([
                    "task: intraday_5m",
                    "tournament_config_path: configs/index_fund_intraday_tournament.example.yml",
                    "summary_output_path: reports/cycle.json",
                    "promotion:",
                    "  registry_dir: model_registry",
                    "  min_balanced_accuracy: 0.35",
                    "  max_regression_mae: 0.12",
                    "  max_high_confidence_accuracy_drop: 0.01",
                    "  max_high_confidence_coverage_drop: 0.02",
                ]),
                encoding="utf-8",
            )

            loaded = load_retraining_cycle_config(config)

        self.assertEqual(loaded.task, "intraday_5m")
        self.assertEqual(loaded.min_balanced_accuracy, 0.35)
        self.assertEqual(loaded.max_regression_mae, 0.12)
        self.assertEqual(loaded.max_high_confidence_accuracy_drop, 0.01)
        self.assertEqual(loaded.max_high_confidence_coverage_drop, 0.02)
        self.assertTrue(str(loaded.tournament_config_path).endswith("configs\\index_fund_intraday_tournament.example.yml") or str(loaded.tournament_config_path).endswith("configs/index_fund_intraday_tournament.example.yml"))


if __name__ == "__main__":
    unittest.main()
