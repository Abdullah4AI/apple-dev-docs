from __future__ import annotations

import re
from pathlib import Path


LINK_RE = re.compile(r'(?<![\w`])!?\[[^\]]*\]\(([^)]+)\)|href="([^"]+)"')
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "tel:", "data:", "javascript:")


def extract_targets(text: str) -> list[str]:
    targets = []
    for match in LINK_RE.finditer(text):
        target = match.group(1) or match.group(2)
        if target:
            targets.append(target.strip())
    return targets


def normalize_target(target: str, *, allow_root_relative: bool) -> str | None:
    target = target.strip()
    if not target or target.startswith("#"):
        return None
    if target.startswith("<") and target.endswith(">"):
        target = target[1:-1]
    if target.startswith("/") and not allow_root_relative:
        return None
    if target.startswith(IGNORED_PREFIXES):
        return None
    target = target.split("#", 1)[0].split("?", 1)[0]
    return target or None


def is_within_root(root: Path, candidate: Path) -> bool:
    try:
        candidate.relative_to(root)
    except ValueError:
        return False
    return True
