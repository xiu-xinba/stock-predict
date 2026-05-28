from __future__ import annotations

import sys
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.onnx_export import default_classifier_onnx_path, export_classifier_sidecar


class OnnxExportUnitTests(unittest.TestCase):
    def test_default_classifier_onnx_path_uses_sidecar_suffix(self) -> None:
        self.assertEqual(
            default_classifier_onnx_path(Path("artifacts/champion.joblib")),
            Path("artifacts/champion.classifier.onnx"),
        )

    def test_export_classifier_sidecar_returns_contract_metadata(self) -> None:
        model = object()
        output = Path("artifacts/champion.classifier.onnx")

        with patch("fund_model_training.onnx_export.export_classifier_to_onnx") as export:
            status = export_classifier_sidecar(model, feature_count=12, output_path=output)

        export.assert_called_once_with(model, 12, output)
        self.assertEqual(status["status"], "ok")
        self.assertEqual(status["path"], str(output))
        self.assertEqual(status["input_name"], "float_input")
        self.assertEqual(status["output_names"], ["label", "probabilities"])
        self.assertEqual(status["purpose"], "direction_classifier_sidecar")

    def test_export_classifier_sidecar_skips_missing_optional_dependency(self) -> None:
        with patch(
            "fund_model_training.onnx_export.export_classifier_to_onnx",
            side_effect=SystemExit("missing skl2onnx"),
        ):
            status = export_classifier_sidecar(object(), feature_count=3, output_path=Path("model.onnx"))

        self.assertEqual(status["status"], "skipped")
        self.assertIn("missing skl2onnx", status["reason"])

    def test_export_classifier_sidecar_can_be_required(self) -> None:
        with patch(
            "fund_model_training.onnx_export.export_classifier_to_onnx",
            side_effect=SystemExit("missing skl2onnx"),
        ):
            with self.assertRaises(SystemExit):
                export_classifier_sidecar(
                    object(),
                    feature_count=3,
                    output_path=Path("model.onnx"),
                    required=True,
                )


if __name__ == "__main__":
    unittest.main()
