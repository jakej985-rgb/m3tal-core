"""Canonical and stable finding fingerprint generator for M3TAL Review Engine."""

import hashlib
import re
from typing import Optional


def normalize_text(text: str) -> str:
    """Normalize text by lowercasing, stripping, and collapsing whitespace."""
    if not text:
        return ""
    text = text.lower().strip()
    text = re.sub(r"[^\w\s]", " ", text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def normalize_file_path(path: str) -> str:
    """Normalize relative file path to canonical POSIX format."""
    if not path:
        return ""
    p = path.strip().replace("\\", "/")
    while p.startswith("./") or p.startswith("/"):
        p = p.lstrip("./").lstrip("/")
    return p.lower()


def generate_finding_fingerprint(
    file_path: str,
    category: str,
    problem_title: str,
    symbol: Optional[str] = None,
    start_line: Optional[int] = None,
) -> str:
    """
    Generate a deterministic SHA256 fingerprint for a finding.
    
    Formula:
      fingerprint = sha256(canonical_path + ":" + category + ":" + anchor + ":" + normalized_title)
    
    Anchor prioritizes symbol/function name (which is stable across line drift).
    If symbol is absent, start_line is used.
    """
    norm_path = normalize_file_path(file_path)
    norm_cat = normalize_text(category)
    norm_title = normalize_text(problem_title)

    if symbol and symbol.strip():
        anchor = f"sym:{normalize_text(symbol)}"
    elif start_line is not None and start_line > 0:
        anchor = f"line:{start_line}"
    else:
        anchor = "global"

    seed = f"{norm_path}:{norm_cat}:{anchor}:{norm_title}"
    return hashlib.sha256(seed.encode("utf-8")).hexdigest()
