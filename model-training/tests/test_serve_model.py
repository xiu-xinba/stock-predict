from __future__ import annotations

import csv
import json
import sys
import threading
import unittest
import urllib.request
from http.server import ThreadingHTTPServer
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.serve_model import ModelServiceConfig, _fund_code_from_path, load_model_service_config, make_handler


class ServeModelUnitTests(unittest.TestCase):
    def test_fund_code_from_model_service_paths(self) -> None:
        self.assertEqual(_fund_code_from_path("/predict/510300"), "510300")
        self.assertEqual(_fund_code_from_path("/api/v1/predict/159915"), "159915")
        self.assertIsNone(_fund_code_from_path("/health"))

    def test_loads_model_config_from_registry_current(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "model.joblib"
            samples = root / "samples.csv"
            current = root / "model_registry" / "daily_index_fund" / "current.json"
            current.parent.mkdir(parents=True)
            model.write_bytes(b"model")
            samples.write_text("fund_code,asof_time\n510300,2026-05-22\n", encoding="utf-8")
            current.write_text(json.dumps({
                "model_path": str(model),
                "samples_path": str(samples),
            }), encoding="utf-8")

            config = load_model_service_config(registry_current_path=current)

        self.assertEqual(config.model_path, model)
        self.assertEqual(config.samples_path, samples)

    def test_loads_prediction_log_output_path(self) -> None:
        config = load_model_service_config(
            model_path=Path("model.joblib"),
            samples_path=Path("samples.csv"),
            prediction_log_path=Path("prediction_log.csv"),
        )

        self.assertEqual(config.prediction_log_path, Path("prediction_log.csv"))

    def test_handler_appends_prediction_log(self) -> None:
        with TemporaryDirectory() as tmp:
            log_path = Path(tmp) / "prediction_log.csv"

            def fake_predict(**_: object) -> dict[str, object]:
                return {
                    "fund_code": "510300",
                    "asof_time": "2026-05-22T14:55:00",
                    "created_at": "2026-05-22T14:55:10",
                    "prediction": {
                        "horizon": "intraday_5m",
                        "predicted_change_pct": 0.12,
                        "direction": "up",
                        "direction_confidence": 0.72,
                        "is_actionable": True,
                    },
                    "model": {
                        "candidate": "logistic_ridge",
                        "model_path": "model.joblib",
                        "feature_set": "index_fund_intraday_v1",
                    },
                }

            config = ModelServiceConfig(
                model_path=Path("model.joblib"),
                samples_path=Path("samples.csv"),
                action_threshold=0.60,
                prediction_log_path=log_path,
            )
            server = ThreadingHTTPServer(("127.0.0.1", 0), make_handler(config, predict_fn=fake_predict))
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                with urllib.request.urlopen(f"http://127.0.0.1:{server.server_port}/predict/510300", timeout=10) as response:
                    payload = json.loads(response.read().decode("utf-8"))
            finally:
                server.shutdown()
                server.server_close()
                thread.join(timeout=5)

            self.assertEqual(payload["prediction"]["direction"], "up")
            with log_path.open(newline="", encoding="utf-8") as handle:
                rows = list(csv.DictReader(handle))

        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["fund_code"], "510300")
        self.assertEqual(rows[0]["horizon"], "intraday_5m")
        self.assertEqual(rows[0]["predicted_direction"], "up")


if __name__ == "__main__":
    unittest.main()
