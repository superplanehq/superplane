import re
from dataclasses import dataclass
from typing import Any, Iterator, Literal

from pydantic_evals.evaluators import EvaluationReason, Evaluator, EvaluatorContext

from ai.models import AddNodeOperation, CanvasAnswer, CanvasOperation

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

        ops = ctx.output.proposal.operations
        add_ops = [op for op in ops if op.type == "add_node"]
        if not add_ops:
            return EvaluationReason(value=False, reason="No add_node operations")

        allowed_names = _effective_canvas_labels(add_ops)
        texts = _collect_config_texts_for_bracket_scan(
            add_ops,
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


def _effective_canvas_labels(add_ops: list[AddNodeOperation]) -> set[str]:
    labels: set[str] = set()
    for op in add_ops:
        name = (op.node_name or "").strip()
        labels.add(name if name else op.block_name)
    return labels


def _collect_config_texts_for_bracket_scan(
    add_ops: list[AddNodeOperation],
    *,
    scan_scope: Literal["all", "last_add_node"],
    target_block_name: str | None,
) -> list[str]:
    candidates: list[AddNodeOperation]
    if target_block_name is not None:
        candidates = [op for op in add_ops if op.block_name == target_block_name]
        if not candidates:
            return []
    elif scan_scope == "last_add_node":
        candidates = [add_ops[-1]]
    else:
        candidates = list(add_ops)

    texts: list[str] = []
    for op in candidates:
        texts.extend(_iter_strings_in_value(op.configuration))
    return texts


def _extract_bracket_keys(text: str) -> list[str]:
    keys = _BRACKET_SINGLE_QUOTED.findall(text) + _BRACKET_DOUBLE_QUOTED.findall(text)
    return [k for k in keys if k]


def iter_config_strings_from_operations(
    operations: list[CanvasOperation],
) -> Iterator[str]:
    for op in operations:
        if op.type == "add_node":
            yield from _iter_strings_in_value(op.configuration)
        elif op.type == "update_node_config":
            yield from _iter_strings_in_value(op.configuration)


def _iter_strings_in_value(value: Any) -> Iterator[str]:
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for nested in value.values():
            yield from _iter_strings_in_value(nested)
    elif isinstance(value, list):
        for item in value:
            yield from _iter_strings_in_value(item)