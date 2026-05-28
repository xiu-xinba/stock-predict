from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.train_tournament import (
    TournamentConfig,
    _ablation_feature_names,
    _evaluate_fitted_models,
    _champion_sort_key,
    _metric_delta,
    _rolling_folds,
    _run_rolling_backtest,
    _split_samples,
)


class TrainTournamentTests(unittest.TestCase):
    def test_champion_sort_prefers_balanced_accuracy(self) -> None:
        weak = {
            "balanced_accuracy": 0.4,
            "classification_accuracy": 0.9,
            "regression_mae": 0.1,
            "high_confidence": {"accuracy": 1.0},
        }
        strong = {
            "balanced_accuracy": 0.5,
            "classification_accuracy": 0.6,
            "regression_mae": 0.5,
            "high_confidence": {"accuracy": 0.5},
        }

        self.assertEqual(sorted([weak, strong], key=_champion_sort_key)[0], strong)

    def test_champion_sort_prefers_actionable_high_confidence_slice(self) -> None:
        no_signal = {
            "balanced_accuracy": 0.5,
            "classification_accuracy": 0.52,
            "regression_mae": 0.8,
            "high_confidence": {"coverage": 0.0, "accuracy": None},
            "calibration": {"ece": 0.03},
            "rolling_backtest": {
                "summary": {
                    "mean_balanced_accuracy": 0.5,
                    "mean_regression_mae": 0.8,
                },
            },
        }
        signal_ready = {
            "balanced_accuracy": 0.48,
            "classification_accuracy": 0.51,
            "regression_mae": 0.85,
            "high_confidence": {"coverage": 0.08, "accuracy": 0.58},
            "calibration": {"ece": 0.06},
            "rolling_backtest": {
                "summary": {
                    "mean_balanced_accuracy": 0.48,
                    "mean_regression_mae": 0.85,
                },
            },
        }

        self.assertEqual(sorted([no_signal, signal_ready], key=_champion_sort_key)[0], signal_ready)

    def test_ablation_feature_names_remove_panic_and_futures_groups(self) -> None:
        features = [
            "fund_return_1d",
            "index_return_1d",
            "futures_return_1d",
            "fear_score",
            "panic_news_component",
        ]

        no_panic = _ablation_feature_names(features, "without_panic_factor")
        no_futures = _ablation_feature_names(features, "without_futures_commodity")

        self.assertNotIn("fear_score", no_panic)
        self.assertNotIn("panic_news_component", no_panic)
        self.assertIn("futures_return_1d", no_panic)
        self.assertNotIn("futures_return_1d", no_futures)
        self.assertIn("fear_score", no_futures)

    def test_metric_delta_handles_missing_values(self) -> None:
        self.assertEqual(_metric_delta(0.45, 0.40), 0.05)
        self.assertIsNone(_metric_delta(None, 0.40))

    def test_intraday_split_purges_overlapping_label_windows(self) -> None:
        samples = pd.DataFrame({
            "fund_code": ["510300"] * 20,
            "asof_time": pd.date_range("2026-05-22 09:30:00", periods=20, freq="min"),
        })
        cfg = TournamentConfig(
            task="intraday",
            data_path=Path("samples.csv"),
            report_output_path=Path("report.json"),
            metadata_output_path=Path("metadata.json"),
            champion_output_path=Path("champion.joblib"),
            feature_set="index_fund_intraday_v1",
            future_return_column="future_return_pct_5m",
            regression_target="future_return_pct_5m",
            test_size=0.3,
            embargo_minutes=5,
        )

        train, test, policy = _split_samples(samples, cfg)

        self.assertEqual(policy["type"], "purged_time_holdout")
        self.assertGreater(policy["removed_train_rows"], 0)
        self.assertLess(train["asof_time"].max() + pd.Timedelta(minutes=5), test["asof_time"].min() - pd.Timedelta(minutes=5))

    def test_evaluation_includes_market_regime_report(self) -> None:
        from sklearn.linear_model import LinearRegression, LogisticRegression

        x = pd.DataFrame({"feature": [0, 1, 2, 3, 4, 5]}, dtype="float32")
        y_cls = pd.Series([0, 0, 1, 1, 2, 2])
        y_reg = pd.Series([-1.0, -0.6, 0.0, 0.2, 0.8, 1.1])
        cls_model = LogisticRegression(max_iter=500).fit(x, y_cls)
        reg_model = LinearRegression().fit(x, y_reg)
        test_df = pd.DataFrame({
            "index_return_5d": [-3, -2, -0.2, 0.1, 2, 3],
            "fear_score": [0.1, 0.2, 0.4, 0.8, 0.9, 1.0],
            "index_volatility_20d": [0.5, 0.6, 0.8, 1.4, 1.8, 2.0],
        })

        report = _evaluate_fitted_models(
            cls_model=cls_model,
            reg_model=reg_model,
            x_test=x,
            y_test_cls=y_cls,
            y_test_reg=y_reg,
            test_df=test_df,
            regression_target="future_return_pct_next_day",
            return_decomposition=None,
            flat_threshold_pct=0.05,
            high_confidence_threshold=0.6,
        )

        self.assertIn("market_regime", report)
        self.assertEqual(report["market_regime"]["groups"]["market_trend"]["status"], "ok")

    def test_rolling_backtest_builds_multiple_walk_forward_folds(self) -> None:
        rows = 45
        samples = pd.DataFrame({
            "fund_code": ["510300"] * rows,
            "asof_time": pd.date_range("2026-01-01", periods=rows, freq="D"),
            "feature": [float(i) for i in range(rows)],
            "label": [i % 3 for i in range(rows)],
            "future_return_pct_next_day": [float((i % 5) - 2) * 0.1 for i in range(rows)],
            "index_return_5d": [float(i % 7) - 3.0 for i in range(rows)],
            "fear_score": [float(i % 10) / 10.0 for i in range(rows)],
            "index_volatility_20d": [1.0 + float(i % 6) / 10.0 for i in range(rows)],
        })
        cfg = TournamentConfig(
            task="rolling",
            data_path=Path("samples.csv"),
            report_output_path=Path("report.json"),
            metadata_output_path=Path("metadata.json"),
            champion_output_path=Path("champion.joblib"),
            feature_set="index_fund_daily_v1",
            candidates=("logistic_ridge",),
            rolling_backtest_folds=3,
            rolling_backtest_min_train_rows=20,
            rolling_backtest_min_test_rows=5,
        )

        report = _run_rolling_backtest(
            samples=samples,
            feature_names=["feature"],
            candidate_name="logistic_ridge",
            cfg=cfg,
        )

        self.assertEqual(report["status"], "ok")
        self.assertEqual(report["summary"]["fold_count"], 3)
        self.assertIn("mean_balanced_accuracy", report["summary"])

    def test_rolling_folds_apply_intraday_purge(self) -> None:
        samples = pd.DataFrame({
            "fund_code": ["510300"] * 40,
            "asof_time": pd.date_range("2026-05-22 09:30:00", periods=40, freq="min"),
        })
        cfg = TournamentConfig(
            task="intraday",
            data_path=Path("samples.csv"),
            report_output_path=Path("report.json"),
            metadata_output_path=Path("metadata.json"),
            champion_output_path=Path("champion.joblib"),
            feature_set="index_fund_intraday_v1",
            future_return_column="future_return_pct_5m",
            regression_target="future_return_pct_5m",
            test_size=0.3,
            embargo_minutes=5,
            rolling_backtest_folds=2,
            rolling_backtest_min_train_rows=10,
            rolling_backtest_min_test_rows=5,
        )

        folds = _rolling_folds(samples, cfg)

        self.assertGreaterEqual(len(folds), 1)
        _, _, metadata = folds[0]
        self.assertTrue(metadata["purged_split"])
        self.assertGreater(metadata["removed_train_rows"], 0)


if __name__ == "__main__":
    unittest.main()
