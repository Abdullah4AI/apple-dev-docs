#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path


def main() -> None:
    repo_root = Path(__file__).resolve().parents[1]
    spec_path = repo_root / "docs" / "openapi" / "latest.json"
    out_path = repo_root / "docs" / "openapi" / "paths.txt"

    if not spec_path.exists():
        raise SystemExit(f"Missing spec file: {spec_path}")

    spec = json.loads(spec_path.read_text())
    paths = spec.get("paths", {})

    methods = {"get", "post", "patch", "delete", "put", "head", "options"}
    lines: list[str] = []

    for path, ops in paths.items():
        for method in ops.keys():
            if method.lower() in methods:
                lines.append(f"{method.upper()} {path}")

    lines.sort()
    out_path.write_text("\n".join(lines) + "\n")
    print(f"Wrote {out_path} ({len(lines)} entries)")


if __name__ == "__main__":
    main()
