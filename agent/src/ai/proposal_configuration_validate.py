"""Validate proposal node configuration against describe_* field schemas (UI parity).

Mirrors web_src/src/pages/workflowv2/applyAiOperationsToWorkflow.ts validation.
"""

from __future__ import annotations

import re
from typing import Any

LIST_LIKE_TYPES = frozenset({"multi-select", "list", "any-predicate-list", "days-of-week"})


def _is_record(value: Any) -> bool:
    return isinstance(value, dict)


def _field_type(field: dict[str, Any]) -> str | None:
    raw = field.get("type")
    return raw if isinstance(raw, str) and raw else None


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


def validate_configuration_by_schema(
    configuration: dict[str, Any],
    fields: list[dict[str, Any]],
    path_prefix: str = "configuration",
) -> list[str]:
    errors: list[str] = []
    fields_by_name: dict[str, dict[str, Any]] = {}
    for item in fields:
        name = item.get("name")
        if isinstance(name, str) and name:
            fields_by_name[name] = item

    for name, value in configuration.items():
        field = fields_by_name.get(name)
        ftype = _field_type(field) if field else None
        if not ftype:
            continue
        errors.extend(validate_configuration_field_value(field, value, f"{path_prefix}.{name}"))

    return errors


def validate_configuration_field_value(
    field: dict[str, Any], value: Any, field_path: str
) -> list[str]:
    ftype = _field_type(field)
    if not ftype:
        return []

    if ftype in LIST_LIKE_TYPES:
        return validate_list_like_configuration_field(field, value, field_path)

    if ftype == "object":
        return validate_object_configuration_field(field, value, field_path)

    if ftype == "number":
        if isinstance(value, bool):
            return [f"{field_path} must be a number"]
        if not isinstance(value, (int, float)):
            return [f"{field_path} must be a number"]

    if ftype == "boolean" and not isinstance(value, bool):
        return [f"{field_path} must be a boolean"]

    return []


def validate_list_like_configuration_field(
    field: dict[str, Any], value: Any, field_path: str
) -> list[str]:
    ftype = _field_type(field) or ""
    if not isinstance(value, list):
        return [f"{field_path} must be an array for field type '{ftype}'"]

    item_schema = _list_item_schema(field)
    if not item_schema:
        return []

    errors: list[str] = []
    for index, item in enumerate(value):
        if not _is_record(item):
            continue
        errors.extend(
            validate_configuration_by_schema(item, item_schema, f"{field_path}[{index}]")
        )
    return errors


def validate_object_configuration_field(
    field: dict[str, Any], value: Any, field_path: str
) -> list[str]:
    if not _is_record(value):
        return [f"{field_path} must be an object"]

    object_schema = _object_schema_fields(field)
    if not object_schema:
        return []

    return validate_configuration_by_schema(value, object_schema, field_path)


# --- Path helpers for deterministic repair (error message parsing) ---

_RE_MUST_OBJECT = re.compile(r"^(.+) must be an object$")
_RE_MUST_NUMBER = re.compile(r"^(.+) must be a number$")
_RE_MUST_BOOLEAN = re.compile(r"^(.+) must be a boolean$")
_RE_MUST_ARRAY = re.compile(r"^(.+) must be an array for field type '([^']+)'$")


def parse_errors_for_repair(messages: list[str]) -> list[tuple[str, str, str | None]]:
    """Parse validation messages into (kind, path, extra) tuples."""
    out: list[tuple[str, str, str | None]] = []
    for msg in messages:
        m = _RE_MUST_OBJECT.match(msg)
        if m:
            out.append(("object", m.group(1), None))
            continue
        m = _RE_MUST_NUMBER.match(msg)
        if m:
            out.append(("number", m.group(1), None))
            continue
        m = _RE_MUST_BOOLEAN.match(msg)
        if m:
            out.append(("boolean", m.group(1), None))
            continue
        m = _RE_MUST_ARRAY.match(msg)
        if m:
            out.append(("array", m.group(1), m.group(2)))
    return out


def configuration_path_to_segments(path: str) -> list[str | int] | None:
    """Turn 'configuration.json' or 'configuration.headers[0].name' into segments."""
    if not path.startswith("configuration"):
        return None
    rest = path[len("configuration") :].lstrip(".")
    if not rest:
        return []

    segments: list[str | int] = []
    i = 0
    length = len(rest)
    while i < length:
        if rest[i] == ".":
            i += 1
            continue
        if rest[i] == "[":
            return None
        j = i
        while j < length and rest[j] not in ".[":
            j += 1
        key = rest[i:j]
        if key:
            segments.append(key)
        if j < length and rest[j] == "[":
            end = rest.index("]", j)
            inner = rest[j + 1 : end].strip()
            try:
                segments.append(int(inner))
            except ValueError:
                return None
            i = end + 1
            continue
        i = j

    return segments


def try_get_value_at_path(root: Any, segments: list[str | int]) -> Any:
    cur: Any = root
    for seg in segments:
        if isinstance(seg, str):
            if not _is_record(cur):
                return None
            cur = cur.get(seg)
        else:
            if not isinstance(cur, list) or seg < 0 or seg >= len(cur):
                return None
            cur = cur[seg]
    return cur


def try_set_value_at_path(root: dict[str, Any], segments: list[str | int], value: Any) -> bool:
    """Assign value at `segments` under dict root; fail if intermediate path is missing."""
    if not segments:
        return False
    cur: Any = root
    for seg in segments[:-1]:
        if isinstance(seg, str):
            if not _is_record(cur) or seg not in cur:
                return False
            cur = cur[seg]
        else:
            if not isinstance(cur, list) or seg < 0 or seg >= len(cur):
                return False
            cur = cur[seg]

    last = segments[-1]
    if isinstance(last, str):
        if not _is_record(cur):
            return False
        cur[last] = value
        return True
    if not isinstance(cur, list) or last < 0 or last >= len(cur):
        return False
    cur[last] = value
    return True
