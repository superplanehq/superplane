"""Coerce, validate, and deterministically repair canvas proposals (UI schema parity)."""

from __future__ import annotations

import copy
import json
import logging
from typing import Any

from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperation,
    CanvasOperationNodeRef,
    CanvasProposal,
    CanvasSummary,
    UpdateNodeConfigOperation,
)
from ai.proposal_configuration_coerce import (
    _cached_block_fields,
    _resolve_block_name_for_update,
    coerce_canvas_answer_proposal,
    coerce_configuration,
)
from ai.proposal_configuration_validate import (
    configuration_path_to_segments,
    parse_errors_for_repair,
    try_get_value_at_path,
    try_set_value_at_path,
    validate_configuration_by_schema,
)
from ai.superplane_client import SuperplaneClient

_LOG = logging.getLogger(__name__)

_MAX_DETERMINISTIC_PASSES = 4


def _is_record(value: Any) -> bool:
    return isinstance(value, dict)


def _resolve_node_id_from_canvas(
    ref: CanvasOperationNodeRef,
    canvas: CanvasSummary | None,
) -> str | None:
    node_id = ref.node_id
    if isinstance(node_id, str) and node_id:
        return node_id
    node_name = ref.node_name
    if canvas is not None and isinstance(node_name, str) and node_name:
        for node in canvas.nodes:
            if node.name == node_name or node.id == node_name:
                return node.id
    return None


def _merged_configuration_for_update(
    client: SuperplaneClient,
    canvas: CanvasSummary | None,
    canvas_id: str | None,
    op: UpdateNodeConfigOperation,
    node_config_cache: dict[str, dict[str, Any]],
) -> dict[str, Any]:
    patch = dict(op.configuration)
    node_id = _resolve_node_id_from_canvas(op.target, canvas)
    if not canvas_id or not node_id:
        return patch
    if node_id not in node_config_cache:
        try:
            details = client.get_node_details(canvas_id, node_id, include_recent_events=False)
            node_config_cache[node_id] = dict(details.configuration)
        except ValueError as exc:
            _LOG.debug("get_node_details for validation merge failed: %s", exc)
            node_config_cache[node_id] = {}
    base = node_config_cache[node_id]
    return {**base, **patch}


def list_proposal_configuration_errors(
    client: SuperplaneClient,
    operations: list[CanvasOperation],
    canvas: CanvasSummary | None,
    schema_cache: dict[str, list[dict[str, Any]] | None],
    node_config_cache: dict[str, dict[str, Any]],
) -> list[tuple[int, str]]:
    """(operation index, error message) for config-bearing ops."""
    canvas_id = canvas.canvas_id if canvas is not None else None
    errors: list[tuple[int, str]] = []
    block_by_node_key: dict[str, str] = {}

    for index, op in enumerate(operations):
        if isinstance(op, AddNodeOperation):
            if op.node_key:
                block_by_node_key[op.node_key] = op.block_name
            fields = _cached_block_fields(client, op.block_name, schema_cache)
            if not fields:
                continue
            merged = dict(op.configuration)
            for msg in validate_configuration_by_schema(merged, fields):
                errors.append((index, msg))
            continue

        if isinstance(op, UpdateNodeConfigOperation):
            block_name = _resolve_block_name_for_update(op, block_by_node_key, canvas)
            if not block_name:
                continue
            fields = _cached_block_fields(client, block_name, schema_cache)
            if not fields:
                continue
            merged = _merged_configuration_for_update(
                client, canvas, canvas_id, op, node_config_cache
            )
            for msg in validate_configuration_by_schema(merged, fields):
                errors.append((index, msg))

    return errors


def _apply_deterministic_repairs_to_config(
    configuration: dict[str, Any],
    messages: list[str],
) -> None:
    """In-place repairs from validation messages (mirrors extra coercion the UI used to do)."""
    for kind, path, field_type in parse_errors_for_repair(messages):
        segments = configuration_path_to_segments(path)
        if segments is None:
            continue
        current = try_get_value_at_path(configuration, segments)

        if kind == "object":
            if isinstance(current, str):
                stripped = current.strip()
                if stripped:
                    try:
                        parsed = json.loads(stripped)
                        if _is_record(parsed):
                            try_set_value_at_path(configuration, segments, parsed)
                            continue
                    except json.JSONDecodeError:
                        pass
            if current is None or not _is_record(current):
                try_set_value_at_path(configuration, segments, {})
            continue

        if kind == "number":
            if isinstance(current, str):
                try:
                    stripped = current.strip()
                    if "." in stripped:
                        try_set_value_at_path(configuration, segments, float(stripped))
                    else:
                        try_set_value_at_path(configuration, segments, int(stripped))
                except ValueError:
                    pass
            continue

        if kind == "boolean":
            if isinstance(current, str):
                low = current.strip().lower()
                if low in ("true", "1"):
                    try_set_value_at_path(configuration, segments, True)
                elif low in ("false", "0"):
                    try_set_value_at_path(configuration, segments, False)
            continue

        if kind == "array":
            if isinstance(current, str):
                try:
                    parsed = json.loads(current)
                    if isinstance(parsed, list):
                        try_set_value_at_path(configuration, segments, parsed)
                        continue
                except json.JSONDecodeError:
                    pass
            if field_type == "multi-select" and isinstance(current, str):
                try_set_value_at_path(configuration, segments, [current])


def _reconcile_operation_config_after_local_repairs(
    client: SuperplaneClient,
    canvas: CanvasSummary | None,
    canvas_id: str | None,
    op: CanvasOperation,
    schema_cache: dict[str, list[dict[str, Any]] | None],
    node_config_cache: dict[str, dict[str, Any]],
    block_by_node_key: dict[str, str],
) -> CanvasOperation:
    if isinstance(op, AddNodeOperation):
        fields = _cached_block_fields(client, op.block_name, schema_cache)
        if not fields:
            return op
        new_config = coerce_configuration(copy.deepcopy(dict(op.configuration)), fields)
        return op.model_copy(update={"configuration": new_config})

    if isinstance(op, UpdateNodeConfigOperation):
        block_name = _resolve_block_name_for_update(op, block_by_node_key, canvas)
        if not block_name:
            return op
        fields = _cached_block_fields(client, block_name, schema_cache)
        if not fields:
            return op
        merged = _merged_configuration_for_update(
            client, canvas, canvas_id, op, node_config_cache
        )
        new_config = coerce_configuration(copy.deepcopy(merged), fields)
        return op.model_copy(update={"configuration": new_config})

    return op


def _repair_operations_locally(
    client: SuperplaneClient,
    operations: list[CanvasOperation],
    canvas: CanvasSummary | None,
    schema_cache: dict[str, list[dict[str, Any]] | None],
    node_config_cache: dict[str, dict[str, Any]],
) -> list[CanvasOperation]:
    """One pass: group errors by op index, mutate configs, re-coerce per op."""
    canvas_id = canvas.canvas_id if canvas is not None else None
    errors = list_proposal_configuration_errors(
        client, operations, canvas, schema_cache, node_config_cache
    )
    if not errors:
        return operations

    by_index: dict[int, list[str]] = {}
    for idx, msg in errors:
        by_index.setdefault(idx, []).append(msg)

    block_by_node_key: dict[str, str] = {}
    for op in operations:
        if isinstance(op, AddNodeOperation) and op.node_key:
            block_by_node_key[op.node_key] = op.block_name

    updated: list[CanvasOperation] = list(operations)

    for index, msgs in by_index.items():
        op = updated[index]
        if isinstance(op, AddNodeOperation):
            fields = _cached_block_fields(client, op.block_name, schema_cache)
            if not fields:
                continue
            cfg = copy.deepcopy(dict(op.configuration))
            _apply_deterministic_repairs_to_config(cfg, msgs)
            updated[index] = op.model_copy(update={"configuration": cfg})
            continue

        if isinstance(op, UpdateNodeConfigOperation):
            block_name = _resolve_block_name_for_update(op, block_by_node_key, canvas)
            if not block_name:
                continue
            fields = _cached_block_fields(client, block_name, schema_cache)
            if not fields:
                continue
            merged = _merged_configuration_for_update(
                client, canvas, canvas_id, op, node_config_cache
            )
            cfg = copy.deepcopy(merged)
            _apply_deterministic_repairs_to_config(cfg, msgs)
            updated[index] = op.model_copy(update={"configuration": cfg})

    reconciled: list[CanvasOperation] = []
    cumulative_keys: dict[str, str] = {}
    for op in updated:
        if isinstance(op, AddNodeOperation) and op.node_key:
            cumulative_keys[op.node_key] = op.block_name
        reconciled.append(
            _reconcile_operation_config_after_local_repairs(
                client,
                canvas,
                canvas_id,
                op,
                schema_cache,
                node_config_cache,
                cumulative_keys,
            )
        )
    return reconciled


def finalize_canvas_proposal_deterministic(
    client: SuperplaneClient,
    proposal: CanvasProposal,
    canvas: CanvasSummary | None,
) -> tuple[CanvasProposal, list[str]]:
    """Coerce and apply deterministic schema repair (API field types).

    Returns ``(normalized_proposal, [])`` when valid, or ``(last_attempt, errors)``.
    """
    temp = CanvasAnswer(
        answer=".",
        confidence=0.5,
        proposal=proposal,
    )
    current = coerce_canvas_answer_proposal(client, temp, canvas)
    if current.proposal is None:
        return proposal, ["proposal_missing_after_coerce"]

    schema_cache: dict[str, list[dict[str, Any]] | None] = {}
    node_config_cache: dict[str, dict[str, Any]] = {}

    operations = list(current.proposal.operations)
    proposal_acc = current.proposal

    for _ in range(_MAX_DETERMINISTIC_PASSES):
        errs = list_proposal_configuration_errors(
            client, operations, canvas, schema_cache, node_config_cache
        )
        if not errs:
            return proposal_acc.model_copy(update={"operations": operations}), []

        operations = _repair_operations_locally(
            client, operations, canvas, schema_cache, node_config_cache
        )
        proposal_acc = proposal_acc.model_copy(update={"operations": operations})
        recoerced = coerce_canvas_answer_proposal(
            client,
            CanvasAnswer(answer=".", confidence=0.5, proposal=proposal_acc),
            canvas,
        )
        if recoerced.proposal is None:
            return proposal_acc, ["proposal_missing_after_coerce"]
        operations = list(recoerced.proposal.operations)
        proposal_acc = recoerced.proposal

    flat_errors = [
        f"[op {idx}] {msg}"
        for idx, msg in list_proposal_configuration_errors(
            client,
            operations,
            canvas,
            schema_cache,
            node_config_cache,
        )
    ]
    return proposal_acc, flat_errors


def apply_deterministic_proposal_finalize_to_answer(
    client: SuperplaneClient,
    answer: CanvasAnswer,
    canvas: CanvasSummary | None,
) -> CanvasAnswer:
    """Post-run safety net: normalize proposal or strip it if still invalid."""
    if answer.proposal is None or not answer.proposal.operations:
        return answer

    normalized, errs = finalize_canvas_proposal_deterministic(
        client, answer.proposal, canvas
    )
    if not errs:
        return answer.model_copy(update={"proposal": normalized})

    note = (
        "\n\nCould not auto-fix every proposed node configuration "
        f"({len(errs)} schema issue(s)). The structured proposal was removed "
        "so Apply is not blocked; retry or adjust nodes manually."
    )
    _LOG.info("post-run proposal validation failed: %s", errs[:10])
    return answer.model_copy(
        update={
            "proposal": None,
            "answer": (answer.answer or "").rstrip() + note,
        }
    )
