from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.run_index_fund_pipeline import load_pipeline_config


class RunIndexFundPipelineTests(unittest.TestCase):
    def test_load_pipeline_config_resolves_paths(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            cfg = root / "pipeline.yml"
            cfg.write_text(
                "\n".join([
                    "universe_path: configs/index_fund_universe.example.csv",
                    "start_date: 20240101",
                    "end_date: 20240523",
                    "raw_dir: data/raw/test",
                    "samples_output_path: data/processed/test.csv",
                ]),
                encoding="utf-8",
            )

            loaded = load_pipeline_config(cfg)

        self.assertTrue(str(loaded.universe_path).endswith("configs\\index_fund_universe.example.csv") or str(loaded.universe_path).endswith("configs/index_fund_universe.example.csv"))
        self.assertEqual(loaded.start_date, "20240101")
        self.assertEqual(loaded.end_date, "20240523")


if __name__ == "__main__":
    unittest.main()
