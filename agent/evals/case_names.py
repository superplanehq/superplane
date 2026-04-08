"""Shared eval case naming (must match across list/filter/logging)."""

from __future__ import annotations

from typing import Any


def eval_case_name(case: Any, index_in_dataset: int) -> str:
    """Return Case.name if set, else ``case_{index_in_dataset}`` in the full dataset order."""
    return getattr(case, "name", f"case_{index_in_dataset}")
