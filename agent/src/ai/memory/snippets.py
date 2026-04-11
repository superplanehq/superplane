"""Derive text from a completed agent run for memory merge input."""

from __future__ import annotations

from typing import Any

from ai.models import CanvasAnswer


def snippet_from_run_output(output: Any) -> str:
    if isinstance(output, CanvasAnswer):
        parts: list[str] = [output.answer.strip()]
        if output.proposal is not None:
            parts.append(f"Proposal summary: {output.proposal.summary.strip()}")
        return "\n".join(parts)
    if isinstance(output, str) and output.strip():
        return output.strip()[:12000]
    if output is None:
        return ""
    text = str(output).strip()
    return text[:12000]
