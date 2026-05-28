from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.rollback_model import rollback_model


class RollbackModelTests(unittest.TestCase):
    def test_restores_rollback_alias_as_current(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            task_dir = root / "registry" / "daily_index_fund"
            task_dir.mkdir(parents=True)
            (task_dir / "current.json").write_text(json.dumps({
                "task": "daily_index_fund",
                "version_id": "v2",
                "model_path": "v2/model.joblib",
            }), encoding="utf-8")
            (task_dir / "rollback.json").write_text(json.dumps({
                "task": "daily_index_fund",
                "version_id": "v1",
                "model_path": "v1/model.joblib",
                "alias": "rollback",
                "aliased_at": "2026-05-25T00:00:00+00:00",
                "replaced_by_version_id": "v2",
            }), encoding="utf-8")
            (task_dir / "rollback.joblib").write_bytes(b"model-v1")

            report = rollback_model("daily_index_fund", registry_dir=root / "registry", reason="test")
            current = json.loads((task_dir / "current.json").read_text(encoding="utf-8"))

            self.assertTrue(report["rolled_back"])
            self.assertEqual(current["version_id"], "v1")
            self.assertEqual(current["rollback_from_version_id"], "v2")
            self.assertNotIn("alias", current)
            self.assertEqual((task_dir / "current.joblib").read_bytes(), b"model-v1")
            self.assertTrue(Path(report["event_path"]).exists())

    def test_missing_rollback_alias_fails(self) -> None:
        with TemporaryDirectory() as tmp:
            with self.assertRaises(FileNotFoundError):
                rollback_model("daily_index_fund", registry_dir=Path(tmp) / "registry")


if __name__ == "__main__":
    unittest.main()
