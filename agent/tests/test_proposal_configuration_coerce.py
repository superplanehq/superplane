"""Unit tests for proposal_configuration_coerce (schema injected; no API)."""

from __future__ import annotations

import json
from typing import Any

import pytest

from ai.proposal_configuration_coerce import coerce_configuration

_ACTIONS_FIELD: dict[str, Any] = {"name": "actions", "type": "multi-select", "type_options": {}}


@pytest.mark.parametrize(
    "raw",
    ['["opened"]', '["opened", "closed"]'],
)
def test_coerce_multi_select_stringified_json_array(raw: str) -> None:
    out = coerce_configuration({"actions": raw}, [_ACTIONS_FIELD])
    assert out["actions"] == json.loads(raw)


def test_coerce_multi_select_already_list() -> None:
    out = coerce_configuration({"actions": ["opened", "closed"]}, [_ACTIONS_FIELD])
    assert out["actions"] == ["opened", "closed"]


def test_coerce_multi_select_plain_string_fallback() -> None:
    out = coerce_configuration({"actions": "opened"}, [_ACTIONS_FIELD])
    assert out["actions"] == ["opened"]


def test_coerce_multi_select_non_string_unchanged() -> None:
    out = coerce_configuration({"actions": 123}, [_ACTIONS_FIELD])
    assert out["actions"] == 123


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("true", True),
        ("false", False),
        ("1", True),
        ("0", False),
        ("TRUE", True),
        (" False ", False),
    ],
)
def test_coerce_boolean_strings(raw: str, expected: bool) -> None:
    field = {"name": "signed", "type": "boolean", "type_options": {}}
    out = coerce_configuration({"signed": raw}, [field])
    assert out["signed"] is expected


def test_coerce_boolean_already_bool() -> None:
    field = {"name": "signed", "type": "boolean", "type_options": {}}
    out = coerce_configuration({"signed": True}, [field])
    assert out["signed"] is True


def test_coerce_boolean_unrecognized_string_unchanged() -> None:
    field = {"name": "signed", "type": "boolean", "type_options": {}}
    out = coerce_configuration({"signed": "maybe"}, [field])
    assert out["signed"] == "maybe"


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("42", 42),
        ("0", 0),
        ("3.5", 3.5),
        ("  10  ", 10),
    ],
)
def test_coerce_number_strings(raw: str, expected: int | float) -> None:
    field = {"name": "n", "type": "number", "type_options": {}}
    out = coerce_configuration({"n": raw}, [field])
    assert out["n"] == expected


def test_coerce_number_invalid_string_unchanged() -> None:
    field = {"name": "n", "type": "number", "type_options": {}}
    out = coerce_configuration({"n": "not_a_number"}, [field])
    assert out["n"] == "not_a_number"


def test_coerce_object_string_with_nested_boolean() -> None:
    outer = {
        "name": "outer",
        "type": "object",
        "type_options": {
            "object": {
                "schema": [
                    {"name": "inner", "type": "boolean", "type_options": {}},
                ]
            }
        },
    }
    out = coerce_configuration({"outer": '{"inner": "true"}'}, [outer])
    assert out["outer"] == {"inner": True}


def test_coerce_object_dict_with_nested_multi_select_string() -> None:
    outer = {
        "name": "outer",
        "type": "object",
        "type_options": {
            "object": {
                "schema": [
                    {"name": "tags", "type": "multi-select", "type_options": {}},
                ]
            }
        },
    }
    out = coerce_configuration({"outer": {"tags": '["a"]'}}, [outer])
    assert out["outer"] == {"tags": ["a"]}


def test_coerce_unknown_keys_preserved() -> None:
    out = coerce_configuration(
        {"actions": '["x"]', "extra": "y"},
        [_ACTIONS_FIELD],
    )
    assert out["actions"] == ["x"]
    assert out["extra"] == "y"


def test_coerce_list_item_schema_nested_boolean() -> None:
    items_field = {
        "name": "items",
        "type": "list",
        "type_options": {
            "list": {
                "itemDefinition": {
                    "schema": [
                        {"name": "enabled", "type": "boolean", "type_options": {}},
                    ]
                }
            }
        },
    }
    out = coerce_configuration(
        {"items": [{"enabled": "false"}, {"enabled": "true"}]},
        [items_field],
    )
    assert out["items"] == [{"enabled": False}, {"enabled": True}]


def test_coerce_list_stringified_array_coerces_item_dicts() -> None:
    items_field = {
        "name": "items",
        "type": "list",
        "type_options": {
            "list": {
                "itemDefinition": {
                    "schema": [
                        {"name": "enabled", "type": "boolean", "type_options": {}},
                    ]
                }
            }
        },
    }
    raw = '[{"enabled":"true"}]'
    out = coerce_configuration({"items": raw}, [items_field])
    assert out["items"] == [{"enabled": True}]


def test_coerce_list_type_plain_string_not_wrapped_like_multi_select() -> None:
    """list (non-multi-select) does not wrap arbitrary string in one-element array."""
    field = {"name": "lst", "type": "list", "type_options": {}}
    out = coerce_configuration({"lst": "opened"}, [field])
    assert out["lst"] == "opened"
