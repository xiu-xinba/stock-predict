from __future__ import annotations

import argparse
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def main() -> None:
    parser = argparse.ArgumentParser(description="Restore model_registry/<task>/rollback.json as the current champion.")
    parser.add_argument("--task", required=True, help="Registry task name, e.g. daily_index_fund or intraday_5m.")
    parser.add_argument("--registry-dir", type=Path, default=Path("model_registry"))
    parser.add_argument("--reason", default="manual rollback")
    args = parser.parse_args()

    report = rollback_model(task=args.task, registry_dir=args.registry_dir, reason=args.reason)
    print(json.dumps(report, ensure_ascii=False, indent=2))


def rollback_model(task: str, registry_dir: str | Path = Path("model_registry"), reason: str = "manual rollback") -> dict[str, Any]:
    registry = Path(registry_dir)
    task_dir = registry / task
    current_path = task_dir / "current.json"
    rollback_path = task_dir / "rollback.json"
    rollback_model_path = task_dir / "rollback.joblib"
    if not rollback_path.exists():
        raise FileNotFoundError(f"rollback alias not found: {rollback_path}")

    current = _read_json(current_path) if current_path.exists() else None
    rollback = _read_json(rollback_path)
    restored = _current_payload_from_rollback(rollback, current=current, reason=reason)
    current_path.write_text(json.dumps(restored, ensure_ascii=False, indent=2), encoding="utf-8")

    restored_model_source = _model_source(rollback, rollback_model_path)
    if restored_model_source:
        shutil.copy2(restored_model_source, task_dir / "current.joblib")

    report = {
        "task": task,
        "rolled_back": True,
        "created_at": restored["rolled_back_at"],
        "reason": reason,
        "restored_version_id": restored.get("version_id"),
        "replaced_current_version_id": current.get("version_id") if current else None,
        "current_path": str(current_path),
        "current_model_path": str(task_dir / "current.joblib") if restored_model_source else None,
    }
    event_dir = task_dir / "rollback_events"
    event_dir.mkdir(parents=True, exist_ok=True)
    event_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    event_path = event_dir / f"{event_id}.json"
    event_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    report["event_path"] = str(event_path)
    return report


def _current_payload_from_rollback(
    rollback: dict[str, Any],
    current: dict[str, Any] | None,
    reason: str,
) -> dict[str, Any]:
    restored = {
        key: value
        for key, value in rollback.items()
        if key not in {"alias", "aliased_at", "replaced_by_version_id"}
    }
    restored["rolled_back_at"] = datetime.now(timezone.utc).isoformat()
    restored["rollback_reason"] = reason
    if current and current.get("version_id"):
        restored["rollback_from_version_id"] = current["version_id"]
    return restored


def _model_source(rollback: dict[str, Any], rollback_model_path: Path) -> Path | None:
    if rollback_model_path.exists():
        return rollback_model_path
    raw = rollback.get("model_path")
    if not raw:
        return None
    source = Path(str(raw))
    return source if source.exists() else None


def _read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


if __name__ == "__main__":
    main()
