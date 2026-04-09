"""Normalize AI proposal configuration to match UI schema validation.

Models often emit stringified JSON for multi-select (e.g. actions) or string
booleans. This module coerces those shapes using the same rules as
``parseDefaultValues`` in web_src (list-like: JSON.parse + multi-select fallback;
boolean/number/object string parsing).
"""

from __future__ import annotations

import json
import logging
from typing import Any

from ai.models import (
    AddNodeOperation,
    CanvasAnswer,
    CanvasOperation,
    CanvasOperationNodeRef,
    CanvasSummary,
    UpdateNodeConfigOperation,
)
from ai.superplane_client import SuperplaneClient

_LOG = logging.getLogger(__name__)

LIST_LIKE_TYPES = frozenset({"multi-select", "list", "any-predicate-list", "days-of-week"})


def _type_options(field: dict[str, Any]) -> dict[str, Any]:
    raw = field.get("type_options") or field.get("typeOptions")
    return raw if isinstance(raw, dict) else {}


def _object_schema_fields(field: dict[str, Any]) -> list[dict[str, Any]]:
    opts = _type_options(field)
    obj = opts.get("object")
    if not isinstance(obj, dict):
        return []
    schema = obj.get("schema")
    return schema if isinstance(schema, list) else []


def _list_item_schema(field: dict[str, Any]) -> list[dict[str, Any]]:
    opts = _type_options(field)
    list_opts = opts.get("list")
    if not isinstance(list_opts, dict):
        return []
    item_def = list_opts.get("itemDefinition") or list_opts.get("item_definition")
    if not isinstance(item_def, dict):
        return []
    schema = item_def.get("schema")
    return schema if isinstance(schema, list) else []


def _coerce_list_like(field_type: str, value: Any) -> Any:
    if isinstance(value, list):
        return value
    if not isinstance(value, str):
        return value
    try:
        parsed = json.loads(value)
        if isinstance(parsed, list):
            return parsed
    except json.JSONDecodeError:
        pass
    if field_type == "multi-select":
        return [value]
    return value


def coerce_value_for_field(field: dict[str, Any], value: Any) -> Any:
    ftype = field.get("type")
    if not isinstance(ftype, str) or not ftype:
        return value

    if ftype in LIST_LIKE_TYPES:
        parsed = _coerce_list_like(ftype, value)
        item_schema = _list_item_schema(field)
        if item_schema and isinstance(parsed, list):
            return [
                coerce_configuration(item, item_schema) if isinstance(item, dict) else item
                for item in parsed
            ]
        return parsed

    if ftype == "boolean" and isinstance(value, str):
        low = value.strip().lower()
        if low == "true":
            return True
        if low == "false":
            return False
        if low == "1":
            return True
        if low == "0":
            return False
        return value

    if ftype == "number" and isinstance(value, str):
        try:
            stripped = value.strip()
            if "." in stripped:
                return float(stripped)
            return int(stripped)
        except ValueError:
            return value

    if ftype == "object":
        obj: Any = value
        if isinstance(value, str):
            try:
                obj = json.loads(value)
            except json.JSONDecodeError:
                return value
        if not isinstance(obj, dict):
            return value
        nested = _object_schema_fields(field)
        if nested:
            return coerce_configuration(obj, nested)
        return obj

    return value


def coerce_configuration(
    configuration: dict[str, Any], fields: list[dict[str, Any]]
) -> dict[str, Any]:
    fields_by_name: dict[str, dict[str, Any]] = {}
    for item in fields:
        name = item.get("name")
        if isinstance(name, str) and name:
            fields_by_name[name] = item

    out: dict[str, Any] = dict(configuration)
    for name, field in fields_by_name.items():
        if name not in out:
            continue
        out[name] = coerce_value_for_field(field, out[name])
    return out


def _fetch_block_configuration_fields(
    client: SuperplaneClient,
    block_name: str,
) -> list[dict[str, Any]] | None:
    try:
        data = client.describe_trigger(block_name)
        fields = data.get("configuration_fields")
        if isinstance(fields, list):
            return fields
    except ValueError:
        pass
    except RuntimeError as exc:
        _LOG.debug("describe_trigger failed for %s: %s", block_name, exc)
        return None

    try:
        data = client.describe_component(block_name)
        fields = data.get("configuration_fields")
        if isinstance(fields, list):
            return fields
    except ValueError:
        return None
    except RuntimeError as exc:
        _LOG.debug("describe_component failed for %s: %s", block_name, exc)
        return None

    return None


def _cached_block_fields(
    client: SuperplaneClient,
    block_name: str,
    cache: dict[str, list[dict[str, Any]] | None],
) -> list[dict[str, Any]] | None:
    if block_name in cache:
        return cache[block_name]
    fields = _fetch_block_configuration_fields(client, block_name)
    cache[block_name] = fields
    return fields


def _block_name_from_canvas(ref: CanvasOperationNodeRef, canvas: CanvasSummary) -> str | None:
    node_id = ref.node_id
    if isinstance(node_id, str) and node_id:
        for node in canvas.nodes:
            if node.id == node_id:
                return node.block_name
    node_name = ref.node_name
    if isinstance(node_name, str) and node_name:
        for node in canvas.nodes:
            if node.name == node_name or node.id == node_name:
                return node.block_name
    return None


def _resolve_block_name_for_update(
    op: UpdateNodeConfigOperation,
    block_by_node_key: dict[str, str],
    canvas: CanvasSummary | None,
) -> str | None:
    key = op.target.node_key
    if isinstance(key, str) and key:
        resolved = block_by_node_key.get(key)
        if resolved:
            return resolved
    if canvas is not None:
        return _block_name_from_canvas(op.target, canvas)
    return None


def _coerce_operations(
    client: SuperplaneClient,
    operations: list[CanvasOperation],
    canvas: CanvasSummary | None,
    schema_cache: dict[str, list[dict[str, Any]] | None],
) -> list[CanvasOperation]:
    block_by_node_key: dict[str, str] = {}
    new_ops: list[CanvasOperation] = []

    for op in operations:
        if isinstance(op, AddNodeOperation):
            if op.node_key:
                block_by_node_key[op.node_key] = op.block_name
            fields = _cached_block_fields(client, op.block_name, schema_cache)
            if fields:
                new_config = coerce_configuration(dict(op.configuration), fields)
                new_ops.append(op.model_copy(update={"configuration": new_config}))
            else:
                new_ops.append(op)
            continue

        if isinstance(op, UpdateNodeConfigOperation):
            block_name = _resolve_block_name_for_update(op, block_by_node_key, canvas)
            if block_name:
                fields = _cached_block_fields(client, block_name, schema_cache)
                if fields:
                    new_config = coerce_configuration(dict(op.configuration), fields)
                    new_ops.append(op.model_copy(update={"configuration": new_config}))
                    continue
            new_ops.append(op)
            continue

        new_ops.append(op)

    return new_ops


def coerce_canvas_answer_proposal(
    client: SuperplaneClient,
    answer: CanvasAnswer,
    canvas: CanvasSummary | None = None,
) -> CanvasAnswer:
    if answer.proposal is None or not answer.proposal.operations:
        return answer

    schema_cache: dict[str, list[dict[str, Any]] | None] = {}
    new_operations = _coerce_operations(
        client,
        list(answer.proposal.operations),
        canvas,
        schema_cache,
    )
    new_proposal = answer.proposal.model_copy(update={"operations": new_operations})
    return answer.model_copy(update={"proposal": new_proposal})
