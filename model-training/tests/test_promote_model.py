from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from fund_model_training.promote_model import PromotionConfig, promote_model


class PromoteModelTests(unittest.TestCase):
    def test_promotes_first_model_that_passes_gates(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            samples = root / "samples.csv"
            model.write_bytes(b"model")
            samples.write_text("fund_code,asof_time\n510300,2026-05-22\n", encoding="utf-8")
            metadata.write_text(json.dumps({
                "data_path": str(samples),
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
            ))

            self.assertTrue(report["promoted"])
            self.assertTrue((root / "registry" / "intraday_5m" / "current.json").exists())
            self.assertTrue((root / "registry" / "intraday_5m" / "current.joblib").exists())
            self.assertEqual(report["samples_path"], str(samples.resolve()))

    def test_rejects_model_below_gate(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.20,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
            ))

            self.assertFalse(report["promoted"])
            self.assertIn("balanced_accuracy below threshold", report["reasons"][0])

    def test_rejects_equal_score_against_current(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")
            cfg = PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
            )

            first = promote_model(cfg)
            second = promote_model(cfg)

            self.assertTrue(first["promoted"])
            self.assertFalse(second["promoted"])
            self.assertIn("candidate score did not improve", second["reasons"][0])

    def test_second_promotion_preserves_previous_current_as_rollback(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model1 = root / "champion-v1.joblib"
            model2 = root / "champion-v2.joblib"
            metadata = root / "metadata.json"
            registry = root / "registry"
            model1.write_bytes(b"model-v1")
            model2.write_bytes(b"model-v2")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            first = promote_model(PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model1,
                challenger_metadata_path=metadata,
                registry_dir=registry,
                min_balanced_accuracy=0.34,
            ))
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.45,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")
            second = promote_model(PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model2,
                challenger_metadata_path=metadata,
                registry_dir=registry,
                min_balanced_accuracy=0.34,
            ))

            self.assertTrue(first["promoted"])
            self.assertTrue(second["promoted"])
            self.assertTrue((registry / "daily_index_fund" / "rollback.json").exists())
            self.assertEqual((registry / "daily_index_fund" / "rollback.joblib").read_bytes(), b"model-v1")
            self.assertEqual(second["rollback"]["version_id"], first["version_id"])

    def test_rejects_score_improvement_below_delta(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")
            cfg = PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
            )
            first = promote_model(cfg)
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.421,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            second = promote_model(PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                min_score_delta=0.002,
            ))

            self.assertTrue(first["promoted"])
            self.assertFalse(second["promoted"])
            self.assertIn("candidate score did not improve", second["reasons"][0])

    def test_rejects_high_confidence_regression_against_current(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.4, "accuracy": 0.7},
                }
            }), encoding="utf-8")
            cfg = PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
            )
            first = promote_model(cfg)
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.50,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.35, "accuracy": 0.65},
                }
            }), encoding="utf-8")

            second = promote_model(PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                max_high_confidence_accuracy_drop=0.0,
                max_high_confidence_coverage_drop=0.0,
            ))

            self.assertTrue(first["promoted"])
            self.assertFalse(second["promoted"])
            self.assertTrue(any("high_confidence.accuracy regressed" in reason for reason in second["reasons"]))
            self.assertTrue(any("high_confidence.coverage regressed" in reason for reason in second["reasons"]))

    def test_rejects_model_with_insufficient_calendar_days(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "train_rows": 800,
                "test_rows": 200,
                "walk_forward": {
                    "train_start": "2026-05-22 09:30:00",
                    "test_end": "2026-05-22 14:55:00",
                },
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                min_train_rows=500,
                min_test_rows=100,
                min_calendar_days=3,
            ))

            self.assertFalse(report["promoted"])
            self.assertTrue(any("sample_days below threshold" in reason for reason in report["reasons"]))

    def test_rejects_model_with_insufficient_sample_days_even_across_weekend(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "train_rows": 800,
                "test_rows": 200,
                "walk_forward": {
                    "train_start": "2026-05-22 09:30:00",
                    "test_end": "2026-05-25 14:55:00",
                    "sample_days": 2,
                },
                "champion_summary": {
                    "balanced_accuracy": 0.42,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="intraday_5m",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                min_train_rows=500,
                min_test_rows=100,
                min_calendar_days=3,
            ))

            self.assertFalse(report["promoted"])
            self.assertTrue(any("sample_days below threshold: 2 < 3" in reason for reason in report["reasons"]))

    def test_rejects_market_regime_weak_slice_when_gate_enabled(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.50,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                    "market_regime": {
                        "groups": {
                            "market_trend": {
                                "status": "ok",
                                "slices": {
                                    "bear": {"rows": 12, "classification_accuracy": 0.25},
                                    "bull": {"rows": 12, "classification_accuracy": 0.60},
                                },
                            },
                        },
                    },
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                min_market_regime_slice_rows=10,
                min_market_regime_accuracy=0.40,
            ))

            self.assertFalse(report["promoted"])
            self.assertTrue(any("market_regime slice accuracy below threshold" in reason for reason in report["reasons"]))

    def test_rejects_weak_rolling_backtest_when_gate_enabled(self) -> None:
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            model = root / "champion.joblib"
            metadata = root / "metadata.json"
            model.write_bytes(b"model")
            metadata.write_text(json.dumps({
                "champion_summary": {
                    "balanced_accuracy": 0.50,
                    "regression_mae": 0.08,
                    "high_confidence": {"coverage": 0.2, "accuracy": 0.7},
                    "rolling_backtest": {
                        "status": "ok",
                        "summary": {
                            "fold_count": 2,
                            "mean_balanced_accuracy": 0.28,
                        },
                    },
                }
            }), encoding="utf-8")

            report = promote_model(PromotionConfig(
                task="daily_index_fund",
                challenger_model_path=model,
                challenger_metadata_path=metadata,
                registry_dir=root / "registry",
                min_balanced_accuracy=0.34,
                min_rolling_backtest_folds=3,
                min_rolling_balanced_accuracy=0.34,
            ))

            self.assertFalse(report["promoted"])
            self.assertTrue(any("rolling_backtest.fold_count below threshold" in reason for reason in report["reasons"]))
            self.assertTrue(any("rolling_backtest.mean_balanced_accuracy below threshold" in reason for reason in report["reasons"]))


if __name__ == "__main__":
    unittest.main()
