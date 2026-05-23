from __future__ import annotations

from pathlib import Path


def export_lightgbm_to_onnx(model, feature_count: int, output_path: str | Path) -> None:
    try:
        import onnx
        import onnxmltools
        from onnxmltools.convert.common.data_types import FloatTensorType
    except ImportError as exc:
        raise SystemExit("Missing ONNX export dependencies. Run `pip install -r requirements.txt`.") from exc

    initial_types = [("float_input", FloatTensorType([None, feature_count]))]
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

    _rename_graph_output(onx, 0, "label")
    _rename_graph_output(onx, 1, "probabilities")

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

    if "float_input" not in input_names:
        raise ValueError(f"Backend contract failed: expected input 'float_input', got {input_names}.")
    if "label" not in output_names or "probabilities" not in output_names:
        raise ValueError(
            "Backend contract failed: expected outputs 'label' and 'probabilities', "
            f"got {output_names}."
        )

    dummy = np.zeros((1, feature_count), dtype=np.float32)
    outputs = dict(zip(output_names, session.run(output_names, {"float_input": dummy})))
    probabilities = outputs["probabilities"]
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
