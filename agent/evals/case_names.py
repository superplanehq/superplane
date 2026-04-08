"""Shared eval case naming (must match across list/filter/logging)."""

from __future__ import annotations

from typing import Any


def eval_case_name(case: Any, index_in_dataset: int) -> str:
    """Return Case.name if truthy, else ``case_{index_in_dataset}`` (matches report.py)."""
    return getattr(case, "name", None) or f"case_{index_in_dataset}"
