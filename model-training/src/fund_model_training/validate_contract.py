from __future__ import annotations

import argparse
import json
from pathlib import Path

from .index_fund_contract import data_dictionary, get_table_spec, known_tables, validate_frame


def main() -> None:
    parser = argparse.ArgumentParser(description="Validate index-fund training CSVs against phase-1 contracts.")
    parser.add_argument("--table", choices=known_tables(), help="Contract table name to validate.")
    parser.add_argument("--csv", type=Path, help="CSV file to validate.")
    parser.add_argument("--emit-dictionary", type=Path, help="Write the full data dictionary as JSON.")
    parser.add_argument("--emit-header", type=Path, help="Write a CSV header template for --table.")
    args = parser.parse_args()

    if args.emit_dictionary:
        args.emit_dictionary.parent.mkdir(parents=True, exist_ok=True)
        args.emit_dictionary.write_text(
            json.dumps(data_dictionary(), ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        print(f"Wrote data dictionary: {args.emit_dictionary}")
        return

    if args.emit_header:
        if not args.table:
            raise SystemExit("--emit-header requires --table.")
        spec = get_table_spec(args.table)
        header = ",".join(column.name for column in spec.columns)
        args.emit_header.parent.mkdir(parents=True, exist_ok=True)
        args.emit_header.write_text(header + "\n", encoding="utf-8")
        print(f"Wrote header template: {args.emit_header}")
        return

    if not args.table or not args.csv:
        raise SystemExit("Provide --table and --csv, or use --emit-dictionary/--emit-header.")

    try:
        import pandas as pd
    except ImportError as exc:
        raise SystemExit("Missing dependency: pandas. Run `pip install -r requirements.txt`.") from exc

    df = pd.read_csv(args.csv)
    errors = validate_frame(args.table, df)
    if errors:
        print(json.dumps({"ok": False, "errors": errors}, ensure_ascii=False, indent=2))
        raise SystemExit(1)
    print(json.dumps({"ok": True, "table": args.table, "rows": int(len(df))}, ensure_ascii=False))


if __name__ == "__main__":
    main()
