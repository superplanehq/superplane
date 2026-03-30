"""JSON-serialization helpers shared across agent packages (e.g. evals).

Distinct from ``repl_web._to_jsonable``: kept separate deliberately so the web
layer can evolve without coupling; this module adds dataclass support for types
such as ``pydantic_ai.usage.RunUsage``.
"""

from __future__ import annotations

import dataclasses
from typing import Any


def to_jsonable(value: Any) -> Any:
    if value is None:
        return None
    if isinstance(value, (str, int, float, bool)):
        return value
    if isinstance(value, dict):
        return {str(key): to_jsonable(item) for key, item in value.items()}
    if isinstance(value, list):
        return [to_jsonable(item) for item in value]
    if isinstance(value, tuple):
        return [to_jsonable(item) for item in value]
    if dataclasses.is_dataclass(value) and not isinstance(value, type):
        return to_jsonable(dataclasses.asdict(value))
    model_dump = getattr(value, "model_dump", None)
    if callable(model_dump):
        return model_dump(mode="json", by_alias=True)
    return str(value)
