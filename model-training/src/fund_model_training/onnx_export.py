from __future__ import annotations

from pathlib import Path
from typing import Any


ONNX_INPUT_NAME = "float_input"
ONNX_LABEL_OUTPUT = "label"
ONNX_PROBABILITY_OUTPUT = "probabilities"


def default_classifier_onnx_path(model_output_path: str | Path) -> Path:
    path = Path(model_output_path)
    if path.suffix:
        return path.with_suffix(".classifier.onnx")
    return path / "classifier.onnx"


def export_classifier_sidecar(
    model,
    feature_count: int,
    output_path: str | Path,
    required: bool = False,
) -> dict[str, Any]:
    """Best-effort ONNX export for the direction classifier in a joblib bundle."""

    path = Path(output_path)
    try:
        export_classifier_to_onnx(model, feature_count, path)
    except SystemExit as exc:
        if required:
            raise
        return _export_status("skipped", path, feature_count, _system_exit_message(exc))
    except Exception as exc:
        if required:
            raise
        return _export_status("failed", path, feature_count, str(exc))
    return _export_status("ok", path, feature_count)


def export_classifier_to_onnx(model, feature_count: int, output_path: str | Path) -> None:
    module = model.__class__.__module__
    if module.startswith("lightgbm."):
        export_lightgbm_to_onnx(model, feature_count, output_path)
        return
    export_sklearn_classifier_to_onnx(model, feature_count, output_path)


def export_lightgbm_to_onnx(model, feature_count: int, output_path: str | Path) -> None:
    try:
        import onnx
        import onnxmltools
        from onnxmltools.convert.common.data_types import FloatTensorType
    except ImportError as exc:
        raise SystemExit("Missing ONNX export dependencies. Run `pip install -r requirements.txt`.") from exc

    initial_types = [(ONNX_INPUT_NAME, FloatTensorType([None, feature_count]))]
    try:
        onx = onnxmltools.convert_lightgbm(
            model,
            initial_types=initial_types,
            target_opset=17,
            zipmap=False,
        )
    except TypeError:
        onx = onnxmltools.convert_lightgbm(
            model,
            initial_types=initial_types,
            target_opset=17,
        )

    _rename_graph_output(onx, 0, ONNX_LABEL_OUTPUT)
    _rename_graph_output(onx, 1, ONNX_PROBABILITY_OUTPUT)

    out = Path(output_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    onnx.save_model(onx, out)
    verify_backend_contract(out, feature_count)


def export_sklearn_classifier_to_onnx(model, feature_count: int, output_path: str | Path) -> None:
    try:
        import onnx
        from skl2onnx import convert_sklearn
        from skl2onnx.common.data_types import FloatTensorType
    except ImportError as exc:
        raise SystemExit("Missing sklearn ONNX export dependencies. Run `pip install -r requirements.txt`.") from exc

    initial_types = [(ONNX_INPUT_NAME, FloatTensorType([None, feature_count]))]
    options = _sklearn_zipmap_options(model)
    try:
        onx = convert_sklearn(
            model,
            initial_types=initial_types,
            target_opset=17,
            options=options,
        )
    except TypeError:
        onx = convert_sklearn(
            model,
            initial_types=initial_types,
            target_opset=17,
        )

    _rename_graph_output(onx, 0, ONNX_LABEL_OUTPUT)
    _rename_graph_output(onx, 1, ONNX_PROBABILITY_OUTPUT)

    out = Path(output_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    onnx.save_model(onx, out)
    verify_backend_contract(out, feature_count)


def verify_backend_contract(model_path: str | Path, feature_count: int) -> None:
    try:
        import numpy as np
        import onnxruntime as ort
    except ImportError as exc:
        raise SystemExit("Missing ONNX runtime dependencies. Run `pip install -r requirements.txt`.") from exc

    session = ort.InferenceSession(str(model_path), providers=["CPUExecutionProvider"])
    input_names = [item.name for item in session.get_inputs()]
    output_names = [item.name for item in session.get_outputs()]

    if ONNX_INPUT_NAME not in input_names:
        raise ValueError(f"Backend contract failed: expected input '{ONNX_INPUT_NAME}', got {input_names}.")
    if ONNX_LABEL_OUTPUT not in output_names or ONNX_PROBABILITY_OUTPUT not in output_names:
        raise ValueError(
            f"Backend contract failed: expected outputs '{ONNX_LABEL_OUTPUT}' and '{ONNX_PROBABILITY_OUTPUT}', "
            f"got {output_names}."
        )

    dummy = np.zeros((1, feature_count), dtype=np.float32)
    outputs = dict(zip(output_names, session.run(output_names, {ONNX_INPUT_NAME: dummy})))
    probabilities = outputs[ONNX_PROBABILITY_OUTPUT]
    if not isinstance(probabilities, np.ndarray):
        raise ValueError("Backend contract failed: 'probabilities' must be a tensor output.")


def _rename_graph_output(model, index: int, new_name: str) -> None:
    if len(model.graph.output) <= index:
        return
    old_name = model.graph.output[index].name
    if old_name == new_name:
        return
    _rename_value(model, old_name, new_name)


def _rename_value(model, old_name: str, new_name: str) -> None:
    for node in model.graph.node:
        for i, value in enumerate(node.input):
            if value == old_name:
                node.input[i] = new_name
        for i, value in enumerate(node.output):
            if value == old_name:
                node.output[i] = new_name
    for collection in (model.graph.input, model.graph.output, model.graph.value_info):
        for value_info in collection:
            if value_info.name == old_name:
                value_info.name = new_name


def _sklearn_zipmap_options(model) -> dict[int, dict[str, bool]]:
    estimator = model
    steps = getattr(model, "steps", None)
    if steps:
        estimator = steps[-1][1]
    return {id(estimator): {"zipmap": False}}


def _export_status(status: str, path: Path, feature_count: int, reason: str | None = None) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "status": status,
        "path": str(path),
        "format": "onnx",
        "feature_count": int(feature_count),
        "input_name": ONNX_INPUT_NAME,
        "output_names": [ONNX_LABEL_OUTPUT, ONNX_PROBABILITY_OUTPUT],
        "purpose": "direction_classifier_sidecar",
    }
    if reason:
        payload["reason"] = reason
    return payload


def _system_exit_message(exc: SystemExit) -> str:
    if exc.code is None:
        return "ONNX export exited without a message."
    return str(exc.code)
