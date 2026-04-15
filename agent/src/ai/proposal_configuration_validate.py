"""Validate AI proposal configuration against component/trigger field schemas.

Runs after coercion to catch values that coercion cannot fix. When the output
validator in ``agent.py`` detects errors it raises ``ModelRetry`` so the LLM
can correct the proposal itself.
"""

from __future__ import annotations

import logging
from typing import Any

from ai.models import (
    AddNodeOperation,
    CanvasOperation,
    CanvasSummary,
    UpdateNodeConfigOperation,
)
from ai.proposal_configuration_coerce import (
    _cached_block_fields,
    _list_item_schema,
    _object_schema_fields,
    _resolve_block_name_for_update,
    _type_options,
)
from ai.superplane_client import SuperplaneClient

_LOG = logging.getLogger(__name__)

_STRING_LIKE_TYPES = frozenset({
    "string",
    "text",
    "expression",
    "xml",
    "time",
    "time-range",
    "date",
    "datetime",
    "day-in-year",
    "cron",
    "timezone",
    "user",
    "role",
    "group",
    "git-ref",
    "secret-key",
})


def _select_options(field: dict[str, Any]) -> list[str] | None:
    opts = _type_options(field)
    select = opts.get("select")
    if not isinstance(select, dict):
        return None
    options = select.get("options")
    if not isinstance(options, list) or not options:
        return None
    return [
        opt["value"]
        for opt in options
        if isinstance(opt, dict) and isinstance(opt.get("value"), str)
    ]


def _multi_select_options(field: dict[str, Any]) -> list[str] | None:
    opts = _type_options(field)
    ms = opts.get("multiSelect") or opts.get("multi_select")
    if not isinstance(ms, dict):
        return None
    options = ms.get("options")
    if not isinstance(options, list) or not options:
        return None
    return [
        opt["value"]
        for opt in options
        if isinstance(opt, dict) and isinstance(opt.get("value"), str)
    ]


def _is_resource_multi(field: dict[str, Any]) -> bool:
    opts = _type_options(field)
    resource = opts.get("resource")
    if not isinstance(resource, dict):
        return False
    return bool(resource.get("multi"))


def validate_value_for_field(field: dict[str, Any], value: Any) -> str | None:
    """Return ``None`` if *value* is valid for *field*, or an error message."""
    ftype = field.get("type")
    if not isinstance(ftype, str) or not ftype:
        return None

    if ftype in _STRING_LIKE_TYPES:
        if not isinstance(value, str):
            return "must be a string"
        return None

    if ftype == "number":
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            return "must be a number"
        return None

    if ftype == "boolean":
        if not isinstance(value, bool):
            return "must be a boolean"
        return None

    if ftype == "select":
        if not isinstance(value, str):
            return "must be a string"
        valid = _select_options(field)
        if valid is not None and value not in valid:
            return f"must be one of: {', '.join(valid)}"
        return None

    if ftype == "multi-select":
        if not isinstance(value, list):
            return "must be a list"
        for item in value:
            if not isinstance(item, str):
                return "all items must be strings"
        valid = _multi_select_options(field)
        if valid is not None:
            valid_set = set(valid)
            for item in value:
                if isinstance(item, str) and item not in valid_set:
                    return f"item '{item}' must be one of: {', '.join(valid)}"
        return None

    if ftype in ("list", "days-of-week", "any-predicate-list"):
        if not isinstance(value, list):
            return "must be a list"
        if ftype == "days-of-week":
            for item in value:
                if not isinstance(item, str):
                    return "all items must be strings"
        if ftype == "list":
            item_schema = _list_item_schema(field)
            if item_schema:
                for i, item in enumerate(value):
                    if isinstance(item, dict):
                        nested = validate_configuration(item, item_schema)
                        if nested:
                            return f"item {i}: {nested[0]}"
        return None

    if ftype == "object":
        nested_schema = _object_schema_fields(field)
        if nested_schema:
            if not isinstance(value, dict):
                return "must be an object"
            nested = validate_configuration(value, nested_schema)
            if nested:
                return nested[0]
            return None
        if not isinstance(value, (dict, list)):
            return "must be an object or array"
        return None

    if ftype == "integration-resource":
        if _is_resource_multi(field):
            if not isinstance(value, list):
                return "must be a list"
            for item in value:
                if not isinstance(item, str):
                    return "all items must be strings"
            return None
        if not isinstance(value, str):
            return "must be a string"
        return None

    return None


def _is_required_by_condition(
    field: dict[str, Any],
    configuration: dict[str, Any],
) -> bool:
    conditions = field.get("required_conditions") or field.get("requiredConditions")
    if not isinstance(conditions, list):
        return False
    for condition in conditions:
        if not isinstance(condition, dict):
            continue
        cond_field = condition.get("field")
        cond_values = condition.get("values")
        if not isinstance(cond_field, str) or not isinstance(cond_values, list):
            continue
        cond_value = configuration.get(cond_field)
        if cond_value is not None and str(cond_value) in [str(v) for v in cond_values]:
            return True
    return False


def validate_configuration(
    configuration: dict[str, Any],
    fields: list[dict[str, Any]],
) -> list[str]:
    """Validate *configuration* against *fields*. Return a list of error messages."""
    errors: list[str] = []
    fields_by_name: dict[str, dict[str, Any]] = {}
    for item in fields:
        name = item.get("name")
        if isinstance(name, str) and name:
            fields_by_name[name] = item

    for name, field in fields_by_name.items():
        value = configuration.get(name)
        is_required = bool(field.get("required"))
        if not is_required:
            is_required = _is_required_by_condition(field, configuration)

        if value is None:
            if is_required:
                errors.append(f"field '{name}' is required")
            continue

        error = validate_value_for_field(field, value)
        if error is not None:
            errors.append(f"field '{name}': {error}")

    return errors


def validate_proposal_operations(
    client: SuperplaneClient,
    operations: list[CanvasOperation],
    canvas: CanvasSummary | None,
) -> list[str]:
    """Validate all operations in a proposal. Return error messages."""
    errors: list[str] = []
    schema_cache: dict[str, list[dict[str, Any]] | None] = {}
    block_by_node_key: dict[str, str] = {}

    for op in operations:
        if isinstance(op, AddNodeOperation):
            if op.node_key:
                block_by_node_key[op.node_key] = op.block_name
            fields = _cached_block_fields(client, op.block_name, schema_cache)
            if fields:
                node_label = op.node_name or op.block_name
                for msg in validate_configuration(dict(op.configuration), fields):
                    errors.append(f"{node_label} ({op.block_name}): {msg}")

        elif isinstance(op, UpdateNodeConfigOperation):
            block_name = _resolve_block_name_for_update(
                op, block_by_node_key, canvas,
            )
            if block_name:
                fields = _cached_block_fields(client, block_name, schema_cache)
                if fields:
                    node_label = op.node_name or block_name
                    for msg in validate_configuration(
                        dict(op.configuration), fields,
                    ):
                        errors.append(f"{node_label} ({block_name}): {msg}")

    return errors
