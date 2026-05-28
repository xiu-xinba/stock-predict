from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import pandas as pd

from fund_model_training.return_decomposition import (
    ensure_return_decomposition_targets,
    fit_return_decomposition,
    predict_return,
    return_decomposition_metadata,
)


class ReturnDecompositionTests(unittest.TestCase):
    def test_ensure_targets_derives_tracking_error(self) -> None:
        samples = pd.DataFrame({
            "future_return_pct_next_day": [1.2, -0.5],
            "future_index_return_pct_next_day": [1.0, -0.2],
        })

        out, columns = ensure_return_decomposition_targets(samples, "future_return_pct_next_day")

        self.assertEqual(columns, ["future_index_return_pct_next_day", "future_tracking_error_pct_next_day"])
        self.assertAlmostEqual(out.loc[0, "future_tracking_error_pct_next_day"], 0.2)
        self.assertAlmostEqual(out.loc[1, "future_tracking_error_pct_next_day"], -0.3)

    def test_predict_return_uses_index_plus_tracking_error_models(self) -> None:
        from sklearn.linear_model import LinearRegression

        x = pd.DataFrame({"feature": [0.0, 1.0, 2.0, 3.0]})
        train = pd.DataFrame({
            "future_return_pct_next_day": [0.3, 1.0, 1.7, 2.4],
            "future_index_return_pct_next_day": [0.2, 0.7, 1.2, 1.7],
            "future_tracking_error_pct_next_day": [0.1, 0.3, 0.5, 0.7],
        })
        direct = LinearRegression().fit(x, train["future_return_pct_next_day"])

        decomposition = fit_return_decomposition(direct, x, train, "future_return_pct_next_day")
        prediction = predict_return(direct, x.iloc[[2]], decomposition)

        self.assertTrue(return_decomposition_metadata(decomposition)["enabled"])
        self.assertEqual(prediction["method"], "tracking_index_plus_error")
        self.assertAlmostEqual(float(prediction["prediction"][0]), 1.7)
        self.assertAlmostEqual(float(prediction["index_return"][0]), 1.2)
        self.assertAlmostEqual(float(prediction["tracking_error"][0]), 0.5)


if __name__ == "__main__":
    unittest.main()
