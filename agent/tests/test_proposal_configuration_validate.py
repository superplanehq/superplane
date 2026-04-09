"""Tests for UI-parity configuration validation and path repair parsing."""

from __future__ import annotations

import pytest

from ai.proposal_configuration_validate import (
    configuration_path_to_segments,
    validate_configuration_by_schema,
)
from ai.proposal_validate_repair import _apply_deterministic_repairs_to_config


def test_validate_object_field_rejects_non_object() -> None:
    field = {
        "name": "json",
        "type": "object",
        "type_options": {"object": {"schema": []}},
    }
    errors = validate_configuration_by_schema({"json": "[]"}, [field])
    assert errors == ["configuration.json must be an object"]


def test_validate_number_rejects_bool() -> None:
    field = {"name": "n", "type": "number", "type_options": {}}
    errors = validate_configuration_by_schema({"n": True}, [field])
    assert errors == ["configuration.n must be a number"]


@pytest.mark.parametrize(
    ("path", "expected"),
    [
        ("configuration.json", ["json"]),
        ("configuration.retry.enabled", ["retry", "enabled"]),
        ("configuration.headers[0].name", ["headers", 0, "name"]),
    ],
)
def test_configuration_path_to_segments(path: str, expected: list) -> None:
    assert configuration_path_to_segments(path) == expected


def test_deterministic_repair_parses_stringified_object() -> None:
    cfg = {"json": '{"a": 1}'}
    _apply_deterministic_repairs_to_config(
        cfg,
        ["configuration.json must be an object"],
    )
    assert cfg == {"json": {"a": 1}}


def test_deterministic_repair_empty_object_for_invalid_shape() -> None:
    cfg = {"json": []}
    _apply_deterministic_repairs_to_config(
        cfg,
        ["configuration.json must be an object"],
    )
    assert cfg == {"json": {}}
