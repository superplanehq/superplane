import re
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any, Literal

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import CanvasAnswer, CanvasChange, CanvasChangeType

_BRACKET_SINGLE_QUOTED = re.compile(r"\$\[\s*'([^']*)'\s*\]")
_BRACKET_DOUBLE_QUOTED = re.compile(r'\$\[\s*"([^"]*)"\s*\]')


@dataclass
class BracketSelectorsMatchCanvasNames(Evaluator):
    """$['…'] / $["…"] keys must match an add_node canvas label (nodeName or blockName)."""

    scan_scope: Literal["all", "last_add_node"] = "all"
    require_at_least_one_selector: bool = True
    target_block_name: str | None = None

    def evaluate(self, ctx: EvaluatorContext[str, CanvasAnswer, Any]) -> EvaluationReason:
        if ctx.output.proposal is None:
            return EvaluationReason(value=False, reason="No proposal to check")

        changes = ctx.output.proposal.changeset.changes or []
        add_changes = [
            change
            for change in changes
            if change.type == CanvasChangeType.ADD_NODE and change.node is not None
        ]
        if not add_changes:
            return EvaluationReason(value=False, reason="No add_node operations")

        allowed_names = _effective_canvas_labels(add_changes)
        texts = _collect_config_texts_for_bracket_scan(
            add_changes,
            scan_scope=self.scan_scope,
            target_block_name=self.target_block_name,
        )
        if not texts and self.scan_scope == "last_add_node":
            return EvaluationReason(value=False, reason="No configuration on last add_node to scan")

        combined = "\n".join(texts)
        keys = _extract_bracket_keys(combined)
        if self.require_at_least_one_selector and not keys:
            return EvaluationReason(
                value=False,
                reason="Expected $['…'] or $[\"…\"] selector in scanned configs",
            )

        for key in keys:
            if key not in allowed_names:
                return EvaluationReason(
                    value=False,
                    reason=(
                        f"Bracket key {key!r} is not a canvas node name "
                        f"(allowed: {sorted(allowed_names)})"
                    ),
                )

        return EvaluationReason(
            value=True,
            reason="All $['…'] / $[\"…\"] keys match canvas node names",
        )


def _effective_canvas_labels(add_changes: list[CanvasChange]) -> set[str]:
    labels: set[str] = set()
    for change in add_changes:
        if change.node is None:
            continue

        name = (change.node.name or "").strip()
        block = (change.node.block or "").strip()
        if name:
            labels.add(name)
            continue
        if block:
            labels.add(block)
    return labels


def _collect_config_texts_for_bracket_scan(
    add_changes: list[CanvasChange],
    *,
    scan_scope: Literal["all", "last_add_node"],
    target_block_name: str | None,
) -> list[str]:
    candidates: list[CanvasChange]
    if target_block_name is not None:
        candidates = [
            change
            for change in add_changes
            if change.node is not None and change.node.block == target_block_name
        ]
        if not candidates:
            return []
    elif scan_scope == "last_add_node":
        candidates = [add_changes[-1]]
    else:
        candidates = list(add_changes)

    texts: list[str] = []
    for change in candidates:
        if change.node is None:
            continue
        if not isinstance(change.node.configuration, dict):
            continue
        texts.extend(_iter_strings_in_value(change.node.configuration))
    return texts


def _extract_bracket_keys(text: str) -> list[str]:
    keys = _BRACKET_SINGLE_QUOTED.findall(text) + _BRACKET_DOUBLE_QUOTED.findall(text)
    return [k for k in keys if k]


def iter_config_strings_from_operations(
    changes: list[CanvasChange],
) -> Iterator[str]:
    for change in changes:
        if change.node is None:
            continue
        if change.type not in (CanvasChangeType.ADD_NODE, CanvasChangeType.UPDATE_NODE):
            continue
        if isinstance(change.node.configuration, dict):
            yield from _iter_strings_in_value(change.node.configuration)


def _iter_strings_in_value(value: Any) -> Iterator[str]:
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for nested in value.values():
            yield from _iter_strings_in_value(nested)
    elif isinstance(value, list):
        for item in value:
            yield from _iter_strings_in_value(item)
