from __future__ import annotations

import argparse
import json
import logging
from dataclasses import dataclass
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Callable
from urllib.parse import parse_qs, unquote, urlparse

from fund_model_training.predict_model import ModelPredictor, predict_from_samples
from fund_model_training.prediction_logging import append_prediction_log


@dataclass(frozen=True)
class ModelServiceConfig:
    model_path: Path
    samples_path: Path
    action_threshold: float
    registry_current_path: Path | None = None
    prediction_log_path: Path | None = None


PredictFn = Callable[..., dict[str, Any]]


def main() -> None:
    parser = argparse.ArgumentParser(description="Serve the trained index-fund champion bundle over HTTP.")
    parser.add_argument("--model", type=Path, help="Champion .joblib bundle.")
    parser.add_argument("--samples", type=Path, help="Processed sample CSV with feature columns.")
    parser.add_argument("--registry-current", type=Path, help="model_registry/<task>/current.json.")
    parser.add_argument("--prediction-log-output", type=Path, help="Optional prediction_log CSV to append on every prediction.")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8090)
    parser.add_argument("--action-threshold", type=float, default=0.60)
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    config = load_model_service_config(
        model_path=args.model,
        samples_path=args.samples,
        registry_current_path=args.registry_current,
        prediction_log_path=args.prediction_log_output,
        action_threshold=args.action_threshold,
    )
    handler = make_handler(config)
    server = ThreadingHTTPServer((args.host, args.port), handler)
    logging.info("Serving fund model at http://%s:%s", args.host, args.port)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logging.info("Stopping fund model service")
    finally:
        server.server_close()


def make_handler(config: ModelServiceConfig, predict_fn: PredictFn = predict_from_samples) -> type[BaseHTTPRequestHandler]:
    predictor = ModelPredictor(config.model_path, config.samples_path) if predict_fn is predict_from_samples else None

    class ModelRequestHandler(BaseHTTPRequestHandler):
        server_version = "FundModelService/0.1"

        def do_GET(self) -> None:  # noqa: N802 - stdlib handler API
            parsed = urlparse(self.path)
            if parsed.path in {"/health", "/api/v1/health"}:
                self._write_json(
                    HTTPStatus.OK,
                    {
                        "status": "ok",
                        "runtime": "python",
                        "model_path": str(config.model_path),
                        "samples_path": str(config.samples_path),
                        "registry_current_path": str(config.registry_current_path) if config.registry_current_path else None,
                        "prediction_log_path": str(config.prediction_log_path) if config.prediction_log_path else None,
                        "loaded_at": predictor.loaded_at.isoformat() if predictor else None,
                        "feature_set": predictor.feature_set if predictor else None,
                        "feature_count": len(predictor.feature_names) if predictor else None,
                    },
                )
                return

            fund_code = _fund_code_from_path(parsed.path)
            if not fund_code:
                self._write_json(HTTPStatus.NOT_FOUND, {"error": "not found"})
                return

            query = parse_qs(parsed.query)
            asof_time = _first_query_value(query, "asof_time")
            try:
                if predictor:
                    payload = predictor.predict(
                        fund_code=fund_code,
                        asof_time=asof_time,
                        action_threshold=config.action_threshold,
                    )
                else:
                    payload = predict_fn(
                        model_path=config.model_path,
                        samples_path=config.samples_path,
                        fund_code=fund_code,
                        asof_time=asof_time,
                        action_threshold=config.action_threshold,
                    )
            except ValueError as exc:
                self._write_json(HTTPStatus.BAD_REQUEST, {"error": str(exc)})
                return
            except Exception as exc:  # pragma: no cover - defensive server boundary
                logging.exception("Prediction failed for fund_code=%s", fund_code)
                self._write_json(HTTPStatus.INTERNAL_SERVER_ERROR, {"error": str(exc)})
                return

            if config.prediction_log_path:
                try:
                    append_prediction_log(payload, config.prediction_log_path, append=True)
                except Exception:  # pragma: no cover - logging must not break inference
                    logging.exception("Failed to append prediction log for fund_code=%s", fund_code)
            self._write_json(HTTPStatus.OK, payload)

        def log_message(self, format: str, *args: Any) -> None:
            logging.info("%s - %s", self.address_string(), format % args)

        def _write_json(self, status: HTTPStatus, payload: dict[str, Any]) -> None:
            body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return ModelRequestHandler


def load_model_service_config(
    model_path: Path | None = None,
    samples_path: Path | None = None,
    registry_current_path: Path | None = None,
    prediction_log_path: Path | None = None,
    action_threshold: float = 0.60,
) -> ModelServiceConfig:
    if registry_current_path:
        current = json.loads(registry_current_path.read_text(encoding="utf-8"))
        if model_path is None and current.get("model_path"):
            model_path = _resolve_registered_path(current["model_path"], registry_current_path)
        if samples_path is None and (current.get("samples_path") or current.get("data_path")):
            samples_path = _resolve_registered_path(current.get("samples_path") or current.get("data_path"), registry_current_path)
    if model_path is None:
        raise SystemExit("--model is required unless --registry-current provides model_path.")
    if samples_path is None:
        raise SystemExit("--samples is required unless --registry-current provides samples_path.")
    return ModelServiceConfig(
        model_path=model_path,
        samples_path=samples_path,
        action_threshold=action_threshold,
        registry_current_path=registry_current_path,
        prediction_log_path=prediction_log_path,
    )


def _resolve_registered_path(value: str | Path, current_path: Path) -> Path:
    path = Path(value)
    if path.is_absolute() or path.exists():
        return path
    project_root = current_path.parent.parent.parent
    candidate = project_root / path
    return candidate if candidate.exists() else path


def _fund_code_from_path(path: str) -> str | None:
    parts = [unquote(part) for part in path.strip("/").split("/") if part]
    if len(parts) == 2 and parts[0] == "predict":
        return parts[1]
    if len(parts) == 4 and parts[:3] == ["api", "v1", "predict"]:
        return parts[3]
    return None


def _first_query_value(query: dict[str, list[str]], key: str) -> str | None:
    values = query.get(key) or []
    return values[0] if values else None


if __name__ == "__main__":
    main()
