from __future__ import annotations

import argparse
import json
import shutil
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class PromotionConfig:
    task: str
    challenger_model_path: Path
    challenger_metadata_path: Path
    registry_dir: Path
    min_balanced_accuracy: float = 0.34
    max_regression_mae: float | None = None
    min_high_confidence_accuracy: float | None = None
    min_high_confidence_coverage: float = 0.0
    min_train_rows: int | None = None
    min_test_rows: int | None = None
    min_calendar_days: int | None = None
    min_score_delta: float = 0.0
    max_high_confidence_accuracy_drop: float | None = None
    max_high_confidence_coverage_drop: float | None = None
    min_market_regime_slice_rows: int | None = None
    min_market_regime_accuracy: float | None = None
    min_rolling_backtest_folds: int | None = None
    min_rolling_balanced_accuracy: float | None = None
    mae_weight: float = 0.05
    allow_lower_score: bool = False


def main() -> None:
    parser = argparse.ArgumentParser(description="Promote a challenger model into the local model registry.")
    parser.add_argument("--task", required=True, help="Registry task name, e.g. daily_index_fund or intraday_5m.")
    parser.add_argument("--model", type=Path, required=True, help="Challenger model artifact.")
    parser.add_argument("--metadata", type=Path, required=True, help="Challenger metadata JSON from train_tournament.")
    parser.add_argument("--registry-dir", type=Path, default=Path("model_registry"))
    parser.add_argument("--min-balanced-accuracy", type=float, default=0.34)
    parser.add_argument("--max-regression-mae", type=float)
    parser.add_argument("--min-high-confidence-accuracy", type=float)
    parser.add_argument("--min-high-confidence-coverage", type=float, default=0.0)
    parser.add_argument("--min-train-rows", type=int)
    parser.add_argument("--min-test-rows", type=int)
    parser.add_argument("--min-calendar-days", type=int)
    parser.add_argument("--min-score-delta", type=float, default=0.0)
    parser.add_argument("--max-high-confidence-accuracy-drop", type=float)
    parser.add_argument("--max-high-confidence-coverage-drop", type=float)
    parser.add_argument("--min-market-regime-slice-rows", type=int)
    parser.add_argument("--min-market-regime-accuracy", type=float)
    parser.add_argument("--min-rolling-backtest-folds", type=int)
    parser.add_argument("--min-rolling-balanced-accuracy", type=float)
    parser.add_argument("--mae-weight", type=float, default=0.05)
    parser.add_argument("--allow-lower-score", action="store_true")
    args = parser.parse_args()

    report = promote_model(PromotionConfig(
        task=args.task,
        challenger_model_path=args.model,
        challenger_metadata_path=args.metadata,
        registry_dir=args.registry_dir,
        min_balanced_accuracy=args.min_balanced_accuracy,
        max_regression_mae=args.max_regression_mae,
        min_high_confidence_accuracy=args.min_high_confidence_accuracy,
        min_high_confidence_coverage=args.min_high_confidence_coverage,
        min_train_rows=args.min_train_rows,
        min_test_rows=args.min_test_rows,
        min_calendar_days=args.min_calendar_days,
        min_score_delta=args.min_score_delta,
        max_high_confidence_accuracy_drop=args.max_high_confidence_accuracy_drop,
        max_high_confidence_coverage_drop=args.max_high_confidence_coverage_drop,
        min_market_regime_slice_rows=args.min_market_regime_slice_rows,
        min_market_regime_accuracy=args.min_market_regime_accuracy,
        min_rolling_backtest_folds=args.min_rolling_backtest_folds,
        min_rolling_balanced_accuracy=args.min_rolling_balanced_accuracy,
        mae_weight=args.mae_weight,
        allow_lower_score=args.allow_lower_score,
    ))
    print(json.dumps(report, ensure_ascii=False, indent=2))


def promote_model(cfg: PromotionConfig) -> dict[str, Any]:
    metadata = json.loads(cfg.challenger_metadata_path.read_text(encoding="utf-8"))
    metrics = metadata.get("champion_summary") or metadata.get("champion") or {}
    if not metrics:
        raise ValueError("metadata does not contain champion_summary metrics")
    samples_path = _samples_path_from_metadata(metadata)
    candidate_score = _score(metrics, cfg.mae_weight)
    current = _load_current(cfg.registry_dir, cfg.task)
    reasons = _gate_reasons(cfg, metadata, metrics, candidate_score, current)
    promoted = not reasons

    report = {
        "task": cfg.task,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "promoted": promoted,
        "reasons": reasons,
        "candidate_score": candidate_score,
        "candidate_metrics": metrics,
        "previous_current": current,
    }
    if promoted:
        version_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
        version_dir = cfg.registry_dir / cfg.task / "versions" / version_id
        version_dir.mkdir(parents=True, exist_ok=True)
        model_path = version_dir / "model.joblib"
        metadata_path = version_dir / "metadata.json"
        promotion_report_path = version_dir / "promotion_report.json"
        shutil.copy2(cfg.challenger_model_path, model_path)
        shutil.copy2(cfg.challenger_metadata_path, metadata_path)
        report.update({
            "version_id": version_id,
            "model_path": str(model_path.resolve()),
            "metadata_path": str(metadata_path.resolve()),
            "samples_path": str(samples_path) if samples_path else None,
            "promotion_report_path": str(promotion_report_path),
        })
        rollback = _write_rollback_alias(cfg.registry_dir, cfg.task, current, version_id, report["created_at"])
        if rollback:
            report["rollback"] = rollback
        promotion_report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
        current_payload = {
            "task": cfg.task,
            "version_id": version_id,
            "model_path": str(model_path.resolve()),
            "metadata_path": str(metadata_path.resolve()),
            "samples_path": str(samples_path) if samples_path else None,
            "metrics": metrics,
            "score": candidate_score,
            "promoted_at": report["created_at"],
        }
        current_path = cfg.registry_dir / cfg.task / "current.json"
        current_path.parent.mkdir(parents=True, exist_ok=True)
        current_path.write_text(json.dumps(current_payload, ensure_ascii=False, indent=2), encoding="utf-8")
        shutil.copy2(model_path, cfg.registry_dir / cfg.task / "current.joblib")
        report["current_path"] = str(current_path)
    return report


def _write_rollback_alias(
    registry_dir: Path,
    task: str,
    current: dict[str, Any] | None,
    replaced_by_version_id: str,
    created_at: str,
) -> dict[str, Any] | None:
    if not current:
        return None
    task_dir = registry_dir / task
    rollback_path = task_dir / "rollback.json"
    rollback_model_path = task_dir / "rollback.joblib"
    payload = {
        **current,
        "alias": "rollback",
        "aliased_at": created_at,
        "replaced_by_version_id": replaced_by_version_id,
    }
    task_dir.mkdir(parents=True, exist_ok=True)
    rollback_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")

    source_model = Path(str(current.get("model_path", "")))
    if not source_model.exists():
        source_model = task_dir / "current.joblib"
    copied_model = False
    if source_model.exists():
        shutil.copy2(source_model, rollback_model_path)
        copied_model = True
    return {
        "path": str(rollback_path),
        "model_path": str(rollback_model_path) if copied_model else None,
        "version_id": current.get("version_id"),
    }


def _gate_reasons(
    cfg: PromotionConfig,
    metadata: dict[str, Any],
    metrics: dict[str, Any],
    candidate_score: float,
    current: dict[str, Any] | None,
) -> list[str]:
    reasons: list[str] = []
    balanced_accuracy = _float(metrics.get("balanced_accuracy"))
    if balanced_accuracy is None or balanced_accuracy < cfg.min_balanced_accuracy:
        reasons.append(f"balanced_accuracy below threshold: {balanced_accuracy} < {cfg.min_balanced_accuracy}")
    regression_mae = _float(metrics.get("regression_mae"))
    if cfg.max_regression_mae is not None and (regression_mae is None or regression_mae > cfg.max_regression_mae):
        reasons.append(f"regression_mae above threshold: {regression_mae} > {cfg.max_regression_mae}")
    high_confidence = metrics.get("high_confidence") or {}
    high_conf_acc = _float(high_confidence.get("accuracy"))
    if cfg.min_high_confidence_accuracy is not None and (
        high_conf_acc is None or high_conf_acc < cfg.min_high_confidence_accuracy
    ):
        reasons.append(
            f"high_confidence.accuracy below threshold: {high_conf_acc} < {cfg.min_high_confidence_accuracy}"
        )
    high_conf_coverage = _float(high_confidence.get("coverage"))
    if high_conf_coverage is None or high_conf_coverage < cfg.min_high_confidence_coverage:
        reasons.append(
            f"high_confidence.coverage below threshold: {high_conf_coverage} < {cfg.min_high_confidence_coverage}"
        )
    train_rows = _int(metadata.get("train_rows"))
    if cfg.min_train_rows is not None and (train_rows is None or train_rows < cfg.min_train_rows):
        reasons.append(f"train_rows below threshold: {train_rows} < {cfg.min_train_rows}")
    test_rows = _int(metadata.get("test_rows"))
    if cfg.min_test_rows is not None and (test_rows is None or test_rows < cfg.min_test_rows):
        reasons.append(f"test_rows below threshold: {test_rows} < {cfg.min_test_rows}")
    sample_days = _sample_days(metadata)
    if cfg.min_calendar_days is not None and (sample_days is None or sample_days < cfg.min_calendar_days):
        reasons.append(f"sample_days below threshold: {sample_days} < {cfg.min_calendar_days}")
    reasons.extend(_rolling_backtest_gate_reasons(cfg, metrics))
    reasons.extend(_market_regime_gate_reasons(cfg, metrics))
    if current:
        current_metrics = current.get("metrics") or {}
        current_high_confidence = current_metrics.get("high_confidence") or {}
        current_high_conf_acc = _float(current_high_confidence.get("accuracy"))
        if cfg.max_high_confidence_accuracy_drop is not None and current_high_conf_acc is not None:
            floor = current_high_conf_acc - cfg.max_high_confidence_accuracy_drop
            if high_conf_acc is None or high_conf_acc < floor:
                reasons.append(
                    "high_confidence.accuracy regressed vs current: "
                    f"{high_conf_acc} < {floor}"
                )
        current_high_conf_coverage = _float(current_high_confidence.get("coverage"))
        if cfg.max_high_confidence_coverage_drop is not None and current_high_conf_coverage is not None:
            floor = current_high_conf_coverage - cfg.max_high_confidence_coverage_drop
            if high_conf_coverage is None or high_conf_coverage < floor:
                reasons.append(
                    "high_confidence.coverage regressed vs current: "
                    f"{high_conf_coverage} < {floor}"
                )
    if current and not cfg.allow_lower_score:
        current_score = _float(current.get("score"))
        required_score = current_score + cfg.min_score_delta
        if current_score is not None and candidate_score <= required_score:
            reasons.append(f"candidate score did not improve: {candidate_score} <= {required_score}")
    return reasons


def _rolling_backtest_gate_reasons(cfg: PromotionConfig, metrics: dict[str, Any]) -> list[str]:
    if cfg.min_rolling_backtest_folds is None and cfg.min_rolling_balanced_accuracy is None:
        return []
    rolling = metrics.get("rolling_backtest") or {}
    summary = rolling.get("summary") or {}
    reasons: list[str] = []
    if rolling.get("status") != "ok":
        reasons.append(f"rolling_backtest not ok: {rolling.get('status') or 'missing'}")
        return reasons
    fold_count = _int(summary.get("fold_count"))
    if cfg.min_rolling_backtest_folds is not None and (
        fold_count is None or fold_count < cfg.min_rolling_backtest_folds
    ):
        reasons.append(f"rolling_backtest.fold_count below threshold: {fold_count} < {cfg.min_rolling_backtest_folds}")
    rolling_balanced_accuracy = _float(summary.get("mean_balanced_accuracy"))
    if cfg.min_rolling_balanced_accuracy is not None and (
        rolling_balanced_accuracy is None or rolling_balanced_accuracy < cfg.min_rolling_balanced_accuracy
    ):
        reasons.append(
            "rolling_backtest.mean_balanced_accuracy below threshold: "
            f"{rolling_balanced_accuracy} < {cfg.min_rolling_balanced_accuracy}"
        )
    return reasons


def _market_regime_gate_reasons(cfg: PromotionConfig, metrics: dict[str, Any]) -> list[str]:
    if cfg.min_market_regime_accuracy is None:
        return []
    min_rows = cfg.min_market_regime_slice_rows or 1
    market_regime = metrics.get("market_regime") or {}
    groups = market_regime.get("groups") or {}
    reasons: list[str] = []
    for group_name, group in groups.items():
        if group.get("status") != "ok":
            continue
        for regime_name, regime_metrics in (group.get("slices") or {}).items():
            rows = _int(regime_metrics.get("rows")) or 0
            if rows < min_rows:
                continue
            accuracy = _float(regime_metrics.get("classification_accuracy"))
            if accuracy is None or accuracy < cfg.min_market_regime_accuracy:
                reasons.append(
                    "market_regime slice accuracy below threshold: "
                    f"{group_name}/{regime_name} {accuracy} < {cfg.min_market_regime_accuracy}"
                )
    return reasons


def _load_current(registry_dir: Path, task: str) -> dict[str, Any] | None:
    current_path = registry_dir / task / "current.json"
    if not current_path.exists():
        return None
    return json.loads(current_path.read_text(encoding="utf-8"))


def _score(metrics: dict[str, Any], mae_weight: float) -> float:
    balanced_accuracy = _float(metrics.get("balanced_accuracy")) or 0.0
    regression_mae = _float(metrics.get("regression_mae")) or 0.0
    high_confidence = metrics.get("high_confidence") or {}
    high_conf_acc = _float(high_confidence.get("accuracy")) or 0.0
    high_conf_coverage = _float(high_confidence.get("coverage")) or 0.0
    return round(balanced_accuracy + high_conf_acc * high_conf_coverage * 0.10 - regression_mae * mae_weight, 8)


def _samples_path_from_metadata(metadata: dict[str, Any]) -> Path | None:
    raw = metadata.get("data_path") or (metadata.get("config") or {}).get("data_path")
    if not raw:
        return None
    return Path(str(raw)).resolve()


def _calendar_days(metadata: dict[str, Any]) -> int | None:
    walk_forward = metadata.get("walk_forward") or {}
    start = _parse_datetime(walk_forward.get("train_start"))
    end = _parse_datetime(walk_forward.get("test_end") or walk_forward.get("train_end"))
    if start is None or end is None:
        return None
    return max((end.date() - start.date()).days + 1, 1)


def _sample_days(metadata: dict[str, Any]) -> int | None:
    walk_forward = metadata.get("walk_forward") or {}
    explicit = _int(walk_forward.get("sample_days") or metadata.get("sample_days"))
    if explicit is not None:
        return explicit
    return _calendar_days(metadata)


def _float(value: Any) -> float | None:
    try:
        if value is None:
            return None
        return float(value)
    except (TypeError, ValueError):
        return None


def _int(value: Any) -> int | None:
    try:
        if value is None:
            return None
        return int(value)
    except (TypeError, ValueError):
        return None


def _parse_datetime(value: Any) -> datetime | None:
    if value is None:
        return None
    raw = str(value).replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(raw)
    except ValueError:
        return None


if __name__ == "__main__":
    main()
