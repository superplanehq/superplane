"""Unit tests for proposal_configuration_validate (schema injected; no API)."""

from __future__ import annotations

from typing import Any

import pytest

from ai.proposal_configuration_validate import (
    validate_configuration,
    validate_value_for_field,
)


def _field(
    name: str = "f",
    ftype: str = "string",
    required: bool = False,
    type_options: dict[str, Any] | None = None,
    **kwargs: Any,
) -> dict[str, Any]:
    out: dict[str, Any] = {"name": name, "type": ftype, "required": required}
    if type_options is not None:
        out["type_options"] = type_options
    out.update(kwargs)
    return out


# ---------------------------------------------------------------------------
# String-like types
# ---------------------------------------------------------------------------


@pytest.mark.parametrize("ftype", ["string", "text", "expression", "xml"])
def test_string_like_accepts_string(ftype: str) -> None:
    assert validate_value_for_field(_field(ftype=ftype), "hello") is None


@pytest.mark.parametrize("ftype", ["string", "text", "expression", "xml"])
def test_string_like_rejects_non_string(ftype: str) -> None:
    assert validate_value_for_field(_field(ftype=ftype), 123) is not None


# ---------------------------------------------------------------------------
# Number
# ---------------------------------------------------------------------------


def test_number_accepts_int() -> None:
    assert validate_value_for_field(_field(ftype="number"), 42) is None


def test_number_accepts_float() -> None:
    assert validate_value_for_field(_field(ftype="number"), 3.5) is None


def test_number_rejects_bool() -> None:
    assert validate_value_for_field(_field(ftype="number"), True) is not None


def test_number_rejects_string() -> None:
    assert validate_value_for_field(_field(ftype="number"), "42") is not None


# ---------------------------------------------------------------------------
# Boolean
# ---------------------------------------------------------------------------


def test_boolean_accepts_true() -> None:
    assert validate_value_for_field(_field(ftype="boolean"), True) is None


def test_boolean_accepts_false() -> None:
    assert validate_value_for_field(_field(ftype="boolean"), False) is None


def test_boolean_rejects_string() -> None:
    assert validate_value_for_field(_field(ftype="boolean"), "true") is not None


def test_boolean_rejects_int() -> None:
    assert validate_value_for_field(_field(ftype="boolean"), 1) is not None


# ---------------------------------------------------------------------------
# Select
# ---------------------------------------------------------------------------


_SELECT_FIELD = _field(
    ftype="select",
    type_options={"select": {"options": [{"label": "GET", "value": "GET"}]}},
)


def test_select_accepts_valid_option() -> None:
    assert validate_value_for_field(_SELECT_FIELD, "GET") is None


def test_select_rejects_invalid_option() -> None:
    err = validate_value_for_field(_SELECT_FIELD, "FETCH")
    assert err is not None
    assert "must be one of" in err


def test_select_rejects_non_string() -> None:
    assert validate_value_for_field(_SELECT_FIELD, 123) is not None


def test_select_without_options_accepts_any_string() -> None:
    assert validate_value_for_field(_field(ftype="select"), "anything") is None


# ---------------------------------------------------------------------------
# Multi-select
# ---------------------------------------------------------------------------


_MULTI_SELECT_FIELD = _field(
    ftype="multi-select",
    type_options={
        "multiSelect": {
            "options": [
                {"label": "A", "value": "a"},
                {"label": "B", "value": "b"},
            ],
        },
    },
)


def test_multi_select_accepts_valid_items() -> None:
    assert validate_value_for_field(_MULTI_SELECT_FIELD, ["a", "b"]) is None


def test_multi_select_rejects_non_list() -> None:
    assert validate_value_for_field(_MULTI_SELECT_FIELD, "a") is not None


def test_multi_select_rejects_non_string_item() -> None:
    assert validate_value_for_field(_MULTI_SELECT_FIELD, [1]) is not None


def test_multi_select_rejects_invalid_option() -> None:
    err = validate_value_for_field(_MULTI_SELECT_FIELD, ["a", "c"])
    assert err is not None
    assert "c" in err


def test_multi_select_without_options_accepts_string_list() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="multi-select"),
            ["x", "y"],
        )
        is None
    )


# ---------------------------------------------------------------------------
# Object
# ---------------------------------------------------------------------------


_OBJECT_WITH_SCHEMA = _field(
    ftype="object",
    type_options={
        "object": {
            "schema": [
                {"name": "enabled", "type": "boolean", "required": False},
            ],
        },
    },
)


def test_object_with_schema_accepts_dict() -> None:
    assert (
        validate_value_for_field(
            _OBJECT_WITH_SCHEMA,
            {"enabled": True},
        )
        is None
    )


def test_object_with_schema_rejects_list() -> None:
    assert validate_value_for_field(_OBJECT_WITH_SCHEMA, [1]) is not None


def test_object_with_schema_rejects_string() -> None:
    assert validate_value_for_field(_OBJECT_WITH_SCHEMA, "text") is not None


def test_object_with_schema_validates_nested() -> None:
    err = validate_value_for_field(
        _OBJECT_WITH_SCHEMA,
        {"enabled": "yes"},
    )
    assert err is not None
    assert "boolean" in err


def test_object_without_schema_accepts_dict() -> None:
    assert validate_value_for_field(_field(ftype="object"), {"a": 1}) is None


def test_object_without_schema_accepts_list() -> None:
    assert validate_value_for_field(_field(ftype="object"), [1, 2]) is None


def test_object_without_schema_rejects_string() -> None:
    assert validate_value_for_field(_field(ftype="object"), "text") is not None


def test_object_without_schema_rejects_number() -> None:
    assert validate_value_for_field(_field(ftype="object"), 42) is not None


# ---------------------------------------------------------------------------
# List
# ---------------------------------------------------------------------------


_LIST_WITH_ITEM_SCHEMA = _field(
    ftype="list",
    type_options={
        "list": {
            "itemDefinition": {
                "schema": [
                    {"name": "key", "type": "string", "required": True},
                ],
            },
        },
    },
)


def test_list_accepts_list() -> None:
    assert (
        validate_value_for_field(
            _LIST_WITH_ITEM_SCHEMA,
            [{"key": "a"}],
        )
        is None
    )


def test_list_rejects_non_list() -> None:
    assert (
        validate_value_for_field(
            _LIST_WITH_ITEM_SCHEMA,
            "not a list",
        )
        is not None
    )


def test_list_validates_item_schema() -> None:
    err = validate_value_for_field(
        _LIST_WITH_ITEM_SCHEMA,
        [{"key": 123}],
    )
    assert err is not None
    assert "item 0" in err


# ---------------------------------------------------------------------------
# Days-of-week
# ---------------------------------------------------------------------------


def test_days_of_week_accepts_string_list() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="days-of-week"),
            ["monday", "friday"],
        )
        is None
    )


def test_days_of_week_rejects_non_list() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="days-of-week"),
            "monday",
        )
        is not None
    )


def test_days_of_week_rejects_non_string_item() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="days-of-week"),
            [1, 2],
        )
        is not None
    )


# ---------------------------------------------------------------------------
# Integration-resource
# ---------------------------------------------------------------------------


def test_integration_resource_single_accepts_string() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="integration-resource"),
            "res-1",
        )
        is None
    )


def test_integration_resource_single_rejects_list() -> None:
    assert (
        validate_value_for_field(
            _field(ftype="integration-resource"),
            ["a"],
        )
        is not None
    )


def test_integration_resource_multi_accepts_list() -> None:
    f = _field(
        ftype="integration-resource",
        type_options={"resource": {"multi": True}},
    )
    assert validate_value_for_field(f, ["a", "b"]) is None


def test_integration_resource_multi_rejects_string() -> None:
    f = _field(
        ftype="integration-resource",
        type_options={"resource": {"multi": True}},
    )
    assert validate_value_for_field(f, "a") is not None


# ---------------------------------------------------------------------------
# Unknown / missing type
# ---------------------------------------------------------------------------


def test_unknown_type_always_valid() -> None:
    assert validate_value_for_field(_field(ftype="custom"), "any") is None


def test_empty_type_always_valid() -> None:
    assert validate_value_for_field({"name": "f"}, "any") is None


# ---------------------------------------------------------------------------
# validate_configuration
# ---------------------------------------------------------------------------


def test_validate_configuration_valid() -> None:
    fields = [
        _field(name="url", ftype="string", required=True),
        _field(name="method", ftype="select"),
    ]
    errors = validate_configuration({"url": "https://x", "method": "GET"}, fields)
    assert errors == []


def test_validate_configuration_required_missing() -> None:
    fields = [_field(name="url", ftype="string", required=True)]
    errors = validate_configuration({}, fields)
    assert len(errors) == 1
    assert "required" in errors[0]


def test_validate_configuration_optional_missing_ok() -> None:
    fields = [_field(name="url", ftype="string", required=False)]
    errors = validate_configuration({}, fields)
    assert errors == []


def test_validate_configuration_type_error() -> None:
    fields = [_field(name="url", ftype="string")]
    errors = validate_configuration({"url": 123}, fields)
    assert len(errors) == 1
    assert "url" in errors[0]


def test_validate_configuration_extra_keys_ignored() -> None:
    fields = [_field(name="url", ftype="string")]
    errors = validate_configuration({"url": "ok", "extra": 999}, fields)
    assert errors == []


def test_validate_configuration_conditional_required() -> None:
    fields = [
        _field(name="method", ftype="select"),
        _field(
            name="json",
            ftype="object",
            required=False,
            required_conditions=[{"field": "method", "values": ["POST"]}],
        ),
    ]
    errors = validate_configuration({"method": "POST"}, fields)
    assert len(errors) == 1
    assert "required" in errors[0]
    assert "json" in errors[0]


def test_validate_configuration_conditional_required_not_met() -> None:
    fields = [
        _field(name="method", ftype="select"),
        _field(
            name="json",
            ftype="object",
            required=False,
            required_conditions=[{"field": "method", "values": ["POST"]}],
        ),
    ]
    errors = validate_configuration({"method": "GET"}, fields)
    assert errors == []


def test_validate_configuration_conditional_required_with_boolean() -> None:
    """Python str(True) gives 'True' but Go gives 'true'; condition values use Go convention."""
    fields = [
        _field(name="enabled", ftype="boolean"),
        _field(
            name="target",
            ftype="string",
            required=False,
            required_conditions=[{"field": "enabled", "values": ["true"]}],
        ),
    ]
    errors = validate_configuration({"enabled": True}, fields)
    assert len(errors) == 1
    assert "required" in errors[0]
    assert "target" in errors[0]
