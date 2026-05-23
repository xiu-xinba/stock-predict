from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class TrainingConfig:
    task: str
    data_path: Path
    model_output_path: Path
    metadata_output_path: Path
    report_output_path: Path
    feature_set: str = "backend_v1"
    label_column: str = "label"
    future_return_column: str = "future_return_pct_next_day"
    flat_threshold_pct: float = 0.05
    test_size: float = 0.2
    random_seed: int = 20260523
    lightgbm: dict[str, Any] = field(default_factory=dict)


def load_config(path: str | Path) -> TrainingConfig:
    try:
        import yaml
    except ImportError as exc:
        raise SystemExit("Missing dependency: PyYAML. Run `pip install -r requirements.txt`.") from exc

    config_path = Path(path)
    with config_path.open("r", encoding="utf-8") as fh:
        raw = yaml.safe_load(fh) or {}

    base_dir = config_path.parent.parent

    def resolve(value: str) -> Path:
        p = Path(value)
        return p if p.is_absolute() else (base_dir / p).resolve()

    return TrainingConfig(
        task=str(raw["task"]),
        data_path=resolve(str(raw["data_path"])),
        model_output_path=resolve(str(raw["model_output_path"])),
        metadata_output_path=resolve(str(raw["metadata_output_path"])),
        report_output_path=resolve(str(raw["report_output_path"])),
        feature_set=str(raw.get("feature_set", "backend_v1")),
        label_column=str(raw.get("label_column", "label")),
        future_return_column=str(raw.get("future_return_column", "future_return_pct_next_day")),
        flat_threshold_pct=float(raw.get("flat_threshold_pct", 0.05)),
        test_size=float(raw.get("test_size", 0.2)),
        random_seed=int(raw.get("random_seed", 20260523)),
        lightgbm=dict(raw.get("lightgbm", {})),
    )
