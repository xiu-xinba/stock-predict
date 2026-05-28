from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import numpy as np

from fund_model_training.collectors.common import require_pandas
from fund_model_training.features import prepare_features
from fund_model_training.prediction_interval import interval_bounds
from fund_model_training.return_decomposition import predict_return, return_decomposition_metadata
from fund_model_training.schema import ID_TO_LABEL


FACTOR_DESCRIPTIONS = {
    "momentum_5d": "基金短期动量",
    "return_10d": "基金十日收益",
    "volatility_20d": "基金二十日波动",
    "volume_ratio": "成交量相对强度",
    "market_beta": "市场联动强度",
    "sector_momentum": "行业/主题动量",
    "flow_signal": "资金流代理信号",
    "mean_reversion": "均值回归信号",
    "fund_return_1d": "基金一日收益",
    "fund_return_5d": "基金五日收益",
    "fund_volatility_20d": "基金二十日波动",
    "index_return_1d": "跟踪指数一日收益",
    "index_return_5d": "跟踪指数五日收益",
    "index_volatility_20d": "跟踪指数波动",
    "fund_tracking_error_1d": "一日跟踪误差",
    "fund_tracking_error_5d": "五日跟踪误差",
    "futures_return_1d": "股指期货一日收益",
    "futures_basis": "股指期货基差",
    "futures_open_interest_change_5d": "期货持仓变化",
    "fear_score": "合成恐慌分数",
    "panic_iv_component": "波动率恐慌分量",
    "panic_flow_component": "资金/持仓恐慌分量",
    "panic_news_component": "下跌/事件恐慌代理",
    "panic_limit_component": "回撤/市场宽度恐慌代理",
    "fund_return_1m": "基金一分钟收益",
    "fund_return_3m": "基金三分钟收益",
    "fund_return_5m": "基金五分钟收益",
    "fund_volatility_15m": "基金十五分钟波动",
    "fund_volume_ratio_20m": "基金二十分钟量比",
    "bid_ask_spread_pct": "买卖价差比例",
    "index_return_1m": "跟踪指数一分钟收益",
    "index_return_3m": "跟踪指数三分钟收益",
    "index_return_5m": "跟踪指数五分钟收益",
    "index_volatility_15m": "跟踪指数十五分钟波动",
    "fund_index_spread_1m": "基金与指数一分钟偏离",
    "fund_index_spread_5m": "基金与指数五分钟偏离",
}

MIN_ACTIONABLE_HIGH_CONFIDENCE_ACCURACY = 0.50
MIN_ACTIONABLE_HIGH_CONFIDENCE_COVERAGE = 0.05
MAX_ACTIONABLE_CALIBRATION_ECE = 0.12


def main() -> None:
    parser = argparse.ArgumentParser(description="Run prediction from a trained index-fund champion bundle.")
    parser.add_argument("--model", type=Path, required=True, help="Champion .joblib bundle.")
    parser.add_argument("--samples", type=Path, required=True, help="Processed sample CSV with feature columns.")
    parser.add_argument("--fund-code", required=True)
    parser.add_argument("--asof-time", help="Optional max asof_time. Uses latest sample at or before this time.")
    parser.add_argument("--output", type=Path, help="Optional JSON output path.")
    parser.add_argument("--action-threshold", type=float, default=0.60)
    args = parser.parse_args()

    result = predict_from_samples(
        model_path=args.model,
        samples_path=args.samples,
        fund_code=args.fund_code,
        asof_time=args.asof_time,
        action_threshold=args.action_threshold,
    )
    payload = json.dumps(result, ensure_ascii=False, indent=2)
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(payload, encoding="utf-8")
    print(payload)


def predict_from_samples(
    model_path: str | Path,
    samples_path: str | Path,
    fund_code: str,
    asof_time: str | None = None,
    action_threshold: float = 0.60,
) -> dict[str, Any]:
    predictor = ModelPredictor(model_path=model_path, samples_path=samples_path)
    return predictor.predict(
        fund_code=fund_code,
        asof_time=asof_time,
        action_threshold=action_threshold,
    )


class ModelPredictor:
    """Loaded joblib model plus prepared feature table for low-latency serving."""

    def __init__(self, model_path: str | Path, samples_path: str | Path) -> None:
        self.model_path = Path(model_path)
        self.samples_path = Path(samples_path)
        self.bundle: dict[str, Any] = {}
        self.feature_names: list[str] = []
        self.feature_set = "index_fund_daily_v1"
        self.regression_target = "future_return_pct_next_day"
        self.raw = None
        self.prepared = None
        self.loaded_at = datetime.now(timezone.utc)
        self.reload()

    def reload(self) -> None:
        pd = require_pandas()
        bundle = _load_model_bundle(self.model_path)
        feature_names = list(bundle["feature_names"])
        config = bundle.get("config", {})
        feature_set = config.get("feature_set", "index_fund_daily_v1")
        raw = pd.read_csv(self.samples_path, dtype={"fund_code": str})
        raw["fund_code"] = raw["fund_code"].astype(str).str.zfill(6)
        raw["asof_time"] = pd.to_datetime(raw["asof_time"], errors="coerce")
        prepared, _ = prepare_features(raw, feature_set)
        prepared["fund_code"] = prepared["fund_code"].astype(str).str.zfill(6)
        prepared["asof_time"] = pd.to_datetime(prepared["asof_time"], errors="coerce")

        self.bundle = bundle
        self.feature_names = feature_names
        self.feature_set = feature_set
        self.regression_target = config.get("regression_target", "future_return_pct_next_day")
        self.raw = raw
        self.prepared = prepared
        self.loaded_at = datetime.now(timezone.utc)

    def predict(
        self,
        fund_code: str,
        asof_time: str | None = None,
        action_threshold: float = 0.60,
    ) -> dict[str, Any]:
        pd = require_pandas()
        if self.raw is None or self.prepared is None:
            raise RuntimeError("ModelPredictor is not loaded.")

        classifier = self.bundle["classifier"]
        regressor = self.bundle["regressor"]
        horizon, target_window = _horizon_from_regression_target(self.regression_target)
        target_code = str(fund_code).zfill(6)
        candidates = self.raw.loc[self.raw["fund_code"] == target_code].dropna(subset=["asof_time"]).copy()
        if asof_time:
            cutoff = pd.to_datetime(asof_time, errors="coerce")
            if pd.isna(cutoff):
                raise ValueError(f"Invalid asof_time: {asof_time}")
            candidates = candidates.loc[candidates["asof_time"] <= cutoff]
        if candidates.empty:
            raise ValueError(f"No sample row found for fund_code={target_code}.")
        candidates = candidates.sort_values("asof_time")
        row = candidates.tail(1).copy()

        sample = self.prepared.loc[
            (self.prepared["fund_code"] == target_code) &
            (self.prepared["asof_time"] == row["asof_time"].iloc[0])
        ].tail(1)
        if sample.empty:
            raise ValueError("Prepared feature row disappeared after feature preparation.")
        x = sample[self.feature_names].astype("float32")

        direction_id = int(classifier.predict(x)[0])
        probabilities = classifier.predict_proba(x)[0] if hasattr(classifier, "predict_proba") else np.array([])
        confidence = float(np.max(probabilities)) if len(probabilities) else 0.0
        return_prediction = predict_return(regressor, x, self.bundle.get("return_decomposition"))
        predicted_return = float(return_prediction["prediction"][0])
        direction = ID_TO_LABEL.get(direction_id, "flat")
        actionability_gate = _bundle_actionability_gate(self.bundle)
        signal_status = _signal_status(direction, confidence, action_threshold, actionability_gate)
        prediction_interval = _prediction_interval_bounds(predicted_return, confidence, self.bundle)
        factors = _top_factors(sample.iloc[0], self.feature_names, self.bundle, limit=6)
        calibration = _bundle_calibration(self.bundle)

        fund_name = str(sample.get("fund_name", pd.Series([target_code])).iloc[0])
        result = {
            "fund_code": target_code,
            "fund_name": fund_name,
            "asof_time": sample["asof_time"].iloc[0].isoformat(),
            "model": {
                "candidate": self.bundle.get("candidate", "unknown"),
                "feature_set": self.feature_set,
                "model_path": str(self.model_path),
                "loaded_at": self.loaded_at.isoformat(),
                "calibration": calibration,
            },
            "feature_snapshot": {
                "feature_set": self.feature_set,
                "features": {
                    feature: _safe_float(sample.iloc[0].get(feature, 0.0))
                    for feature in self.feature_names
                },
            },
            "prediction": {
                "horizon": horizon,
                "target_window": target_window,
                "direction": direction,
                "direction_confidence": round(confidence, 4),
                "predicted_change_pct": round(predicted_return, 4),
                "return_decomposition": _return_decomposition_payload(return_prediction, self.bundle.get("return_decomposition")),
                "change_range": {
                    "low": prediction_interval["low"],
                    "high": prediction_interval["high"],
                },
                "prediction_interval": prediction_interval,
                "class_probabilities": _probability_payload(probabilities, getattr(classifier, "classes_", None)),
                "top_factors": factors,
                "actionability_gate": actionability_gate,
                "signal_status": signal_status,
                "is_actionable": signal_status == "actionable",
                "reliability": _reliability_from_calibration(calibration),
                "reliability_note": _reliability_note(calibration),
            },
            "data_quality": {
                "feature_count": len(self.feature_names),
                "sample_rows": int(len(self.prepared)),
                "latest_sample_time": _latest_asof_time(self.prepared),
                "sample_source": "processed_csv",
                "has_panic_factor": bool(float(sample.get("fear_score", 0.0).iloc[0]) != 0.0),
                "has_futures_features": bool(float(sample.get("futures_return_1d", 0.0).iloc[0]) != 0.0),
                "note": "基于 processed samples 的最近可用特征行推理；模型和样本在服务启动时加载并复用。",
            },
            "created_at": datetime.now(timezone.utc).isoformat(),
        }
        return result


def _load_model_bundle(model_path: str | Path) -> dict[str, Any]:
    try:
        import joblib
    except ImportError as exc:
        raise SystemExit("Missing dependency: joblib. Run `pip install joblib`.") from exc

    return joblib.load(model_path)


def _bundle_calibration(bundle: dict[str, Any]) -> dict[str, Any] | None:
    report = bundle.get("champion_report") or bundle.get("report") or {}
    calibration = report.get("calibration")
    if not calibration:
        return None
    return {
        "ece": calibration.get("ece"),
        "mce": calibration.get("mce"),
        "brier_score": calibration.get("brier_score"),
    }


def _bundle_prediction_interval(bundle: dict[str, Any]) -> dict[str, Any] | None:
    interval = bundle.get("prediction_interval")
    if interval:
        return interval
    report = bundle.get("champion_report") or bundle.get("report") or {}
    interval = report.get("prediction_interval")
    return interval or None


def _bundle_actionability_gate(
    bundle: dict[str, Any],
    min_high_confidence_accuracy: float = MIN_ACTIONABLE_HIGH_CONFIDENCE_ACCURACY,
    min_high_confidence_coverage: float = MIN_ACTIONABLE_HIGH_CONFIDENCE_COVERAGE,
    max_calibration_ece: float = MAX_ACTIONABLE_CALIBRATION_ECE,
) -> dict[str, Any]:
    report = bundle.get("champion_report") or bundle.get("report") or {}
    high_confidence = report.get("high_confidence") or {}
    high_confidence_accuracy = _number_or_none(high_confidence.get("accuracy"))
    high_confidence_coverage = _number_or_none(high_confidence.get("coverage"))
    if high_confidence_accuracy is None:
        return {
            "actionable": False,
            "reason": "missing_high_confidence_accuracy",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": None,
            "high_confidence_coverage": high_confidence_coverage,
        }
    if high_confidence_accuracy < min_high_confidence_accuracy:
        return {
            "actionable": False,
            "reason": "high_confidence_accuracy_below_threshold",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": round(high_confidence_accuracy, 6),
            "high_confidence_coverage": high_confidence_coverage,
        }
    if high_confidence_coverage is None:
        return {
            "actionable": False,
            "reason": "missing_high_confidence_coverage",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": round(high_confidence_accuracy, 6),
            "high_confidence_coverage": None,
        }
    if high_confidence_coverage < min_high_confidence_coverage:
        return {
            "actionable": False,
            "reason": "high_confidence_coverage_below_threshold",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": round(high_confidence_accuracy, 6),
            "high_confidence_coverage": round(high_confidence_coverage, 6),
        }

    calibration_ece = _number_or_none((report.get("calibration") or {}).get("ece"))
    if calibration_ece is None:
        return {
            "actionable": False,
            "reason": "missing_calibration_ece",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": round(high_confidence_accuracy, 6),
            "high_confidence_coverage": round(high_confidence_coverage, 6),
            "max_calibration_ece": max_calibration_ece,
            "calibration_ece": None,
        }
    if calibration_ece > max_calibration_ece:
        return {
            "actionable": False,
            "reason": "calibration_ece_above_threshold",
            "min_high_confidence_accuracy": min_high_confidence_accuracy,
            "min_high_confidence_coverage": min_high_confidence_coverage,
            "high_confidence_accuracy": round(high_confidence_accuracy, 6),
            "high_confidence_coverage": round(high_confidence_coverage, 6),
            "max_calibration_ece": max_calibration_ece,
            "calibration_ece": round(calibration_ece, 6),
        }

    return {
        "actionable": True,
        "reason": "passed",
        "min_high_confidence_accuracy": min_high_confidence_accuracy,
        "min_high_confidence_coverage": min_high_confidence_coverage,
        "high_confidence_accuracy": round(high_confidence_accuracy, 6),
        "high_confidence_coverage": round(high_confidence_coverage, 6),
        "max_calibration_ece": max_calibration_ece,
        "calibration_ece": round(calibration_ece, 6),
    }


def _reliability_from_calibration(calibration: dict[str, Any] | None) -> str:
    if not calibration or calibration.get("ece") is None:
        return "model_mvp"
    try:
        ece = float(calibration["ece"])
    except (TypeError, ValueError):
        return "model_mvp"
    if ece <= 0.05:
        return "model_calibrated"
    if ece <= 0.12:
        return "model_calibration_watch"
    return "model_uncalibrated"


def _reliability_note(calibration: dict[str, Any] | None) -> str:
    if not calibration or calibration.get("ece") is None:
        return "公开数据 MVP 模型；已跑通训练和回测流程，但尚未写入概率校准指标。"
    ece = calibration.get("ece")
    return f"公开数据 MVP 模型；训练报告记录 ECE={ece}，高置信信号仍需影子验证和真实交易成本评估。"


def _latest_asof_time(prepared) -> str | None:
    latest = prepared["asof_time"].max()
    if latest is None:
        return None
    try:
        if require_pandas().isna(latest):
            return None
    except TypeError:
        return None
    return latest.isoformat()


def _prediction_spread(predicted_return: float, confidence: float) -> float:
    base = 0.35 + (1.0 - min(max(confidence, 0.0), 1.0)) * 1.2
    return max(base, abs(predicted_return) * 0.35)


def _prediction_interval_bounds(predicted_return: float, confidence: float, bundle: dict[str, Any]) -> dict[str, Any]:
    fallback = _prediction_spread(predicted_return, confidence)
    return interval_bounds(
        predicted_return=predicted_return,
        report=_bundle_prediction_interval(bundle),
        fallback_spread=fallback,
    )


def _horizon_from_regression_target(regression_target: str) -> tuple[str, str]:
    if regression_target == "future_return_pct_5m":
        return "intraday_5m", "未来5分钟"
    if regression_target == "future_return_pct_3m":
        return "intraday_3m", "未来3分钟"
    if regression_target == "future_return_pct_1w":
        return "next_week", "未来一周"
    return "next_day", "下一个交易日"


def _signal_status(
    direction: str,
    confidence: float,
    action_threshold: float,
    actionability_gate: dict[str, Any] | None = None,
) -> str:
    if confidence < action_threshold:
        return "low_confidence"
    if direction == "flat":
        return "no_signal"
    if actionability_gate is not None and not bool(actionability_gate.get("actionable")):
        return "low_confidence"
    return "actionable"


def _probability_payload(probabilities, classes=None) -> dict[str, float]:
    if probabilities is None or len(probabilities) == 0:
        return {}
    if classes is None:
        classes = list(range(len(probabilities)))
    payload = {label: 0.0 for label in ID_TO_LABEL.values()}
    for raw_class, value in zip(classes, probabilities):
        try:
            class_id = int(raw_class)
        except (TypeError, ValueError):
            label = str(raw_class)
        else:
            label = ID_TO_LABEL.get(class_id, str(class_id))
        payload[label] = round(float(value), 6)
    return payload


def _return_decomposition_payload(return_prediction: dict[str, Any], decomposition: dict[str, Any] | None) -> dict[str, Any]:
    metadata = return_decomposition_metadata(decomposition)
    payload = {
        "enabled": bool(metadata.get("enabled")),
        "method": return_prediction.get("method", metadata.get("method")),
        "formula": "fund_return = tracking_index_return + tracking_error" if metadata.get("enabled") else "fund_return = direct_model_output",
        "index_return_pct": _first_prediction_value(return_prediction.get("index_return")),
        "tracking_error_pct": _first_prediction_value(return_prediction.get("tracking_error")),
        "direct_fund_return_pct": _first_prediction_value(return_prediction.get("direct_fund_return")),
    }
    if metadata.get("index_return_target"):
        payload["index_return_target"] = metadata["index_return_target"]
    if metadata.get("tracking_error_target"):
        payload["tracking_error_target"] = metadata["tracking_error_target"]
    return payload


def _first_prediction_value(values) -> float | None:
    if values is None:
        return None
    try:
        return _safe_float(values[0])
    except (TypeError, IndexError, KeyError):
        return None


def _top_factors(sample_row, feature_names: list[str], bundle: dict[str, Any], limit: int = 6) -> list[dict[str, Any]]:
    importances = _feature_importances(bundle, feature_names)
    values = []
    for feature in feature_names:
        raw_value = sample_row.get(feature, 0.0)
        try:
            numeric = abs(float(raw_value))
        except (TypeError, ValueError):
            numeric = 0.0
        score = numeric * importances.get(feature, 1.0)
        values.append((feature, score, raw_value))
    values.sort(key=lambda item: item[1], reverse=True)
    total = sum(score for _, score, _ in values[:limit]) or 1.0
    return [
        {
            "name": feature,
            "importance": round(float(score / total), 4),
            "value": _safe_float(raw_value),
            "description": FACTOR_DESCRIPTIONS.get(feature, feature),
        }
        for feature, score, raw_value in values[:limit]
    ]


def _feature_importances(bundle: dict[str, Any], feature_names: list[str]) -> dict[str, float]:
    model = bundle.get("classifier")
    raw = getattr(model, "feature_importances_", None)
    if raw is None:
        return {feature: 1.0 for feature in feature_names}
    total = float(np.sum(np.abs(raw))) or 1.0
    return {
        feature: max(float(abs(value) / total), 0.0001)
        for feature, value in zip(feature_names, raw)
    }


def _safe_float(value) -> float | None:
    try:
        if value is None:
            return None
        numeric = float(value)
        if np.isnan(numeric) or np.isinf(numeric):
            return None
        return round(numeric, 6)
    except (TypeError, ValueError):
        return None


def _number_or_none(value: Any) -> float | None:
    try:
        if value is None:
            return None
        numeric = float(value)
    except (TypeError, ValueError):
        return None
    if np.isnan(numeric) or np.isinf(numeric):
        return None
    return numeric


if __name__ == "__main__":
    main()
