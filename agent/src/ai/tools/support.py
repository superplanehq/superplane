from typing import Any

from ai.config import config


def tool_failure(
    tool_name: str,
    message: str,
    *,
    code: str | None = None,
    **context: Any,
) -> dict[str, Any]:
    """Uniform tool error / failure dict for the model (and logging).

    Always includes ``__tool_error__`` (human-readable message) and
    ``__tool_name__``. Optional ``__tool_error_code__`` for stable categories.
    Extra keyword arguments are copied when not None (e.g. pattern_id, name).
    """
    payload: dict[str, Any] = {
        "__tool_error__": message,
        "__tool_name__": tool_name,
    }
    if code is not None:
        payload["__tool_error_code__"] = code
    for key, value in context.items():
        if value is not None:
            payload[key] = value
    return payload


def tool_error_entry(tool_name: str, error: Exception) -> dict[str, Any]:
    return tool_failure(tool_name, str(error))


def tool_debug(message: str) -> None:
    if config.debug:
        print(f"[web][agent] {message}", flush=True)
