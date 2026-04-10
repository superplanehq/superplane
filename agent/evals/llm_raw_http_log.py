"""Log raw LLM HTTP traffic (Anthropic, etc.) into eval case logs.

Patches :func:`pydantic_ai.models._cached_async_http_client` once so every
``httpx.AsyncClient`` used for provider ``'anthropic'`` (and the default pool)
gets an async response hook that writes full request/response bodies to the
active case log (see :data:`evals.otel_case_log.current_eval_case_name`).

On by default for eval runs. Set ``EVAL_LLM_HTTP_RAW=0`` to disable. Authorization
headers are redacted in the log.
"""

from __future__ import annotations

import functools
import json
import os
from typing import Any

import httpx

from evals.case_logger import CaseLogger
from evals.otel_case_log import current_eval_case_name

_INSTALLED = False

_REDACT_HEADERS = frozenset({"authorization", "x-api-key"})


def _eval_llm_http_raw_enabled() -> bool:
    value = os.getenv("EVAL_LLM_HTTP_RAW", "1").strip().lower()
    return value not in ("0", "false", "no")


def _safe_headers(headers: httpx.Headers) -> dict[str, str]:
    out: dict[str, str] = {}
    for key, value in headers.items():
        if key.lower() in _REDACT_HEADERS:
            out[key] = "<redacted>"
        else:
            out[key] = value
    return out


def _format_body_preview(content: bytes) -> str:
    try:
        text = content.decode("utf-8")
    except UnicodeDecodeError:
        return f"<{len(content)} bytes non-utf8>"
    try:
        parsed: Any = json.loads(text)
        return json.dumps(parsed, indent=2, ensure_ascii=False)
    except json.JSONDecodeError:
        return text


async def _log_response(case_logger: CaseLogger, response: httpx.Response) -> None:
    case_name = current_eval_case_name.get()
    if case_name is None:
        return
    await response.aread()
    request = response.request
    req_headers = _safe_headers(request.headers)
    req_body = _format_body_preview(request.content)
    resp_body = _format_body_preview(response.content)
    block = (
        "LLM_HTTP_RAW\n"
        f"request {request.method} {request.url}\n"
        f"request_headers={json.dumps(req_headers, indent=2, ensure_ascii=False)}\n"
        "request_body=\n"
        f"{req_body}\n"
        f"response_status={response.status_code}\n"
        "response_body=\n"
        f"{resp_body}\n"
    )
    case_logger.log_case_sync(case_name, block)


def install_llm_raw_http_logging(case_logger: CaseLogger) -> None:
    """Install httpx hooks on pydantic-ai's cached async HTTP clients (idempotent)."""

    global _INSTALLED  # noqa: PLW0603
    if _INSTALLED or not _eval_llm_http_raw_enabled():
        return

    from pydantic_ai.models import (  # noqa: PLC0415
        DEFAULT_HTTP_TIMEOUT,
        _cached_async_http_client,
        get_user_agent,
    )

    _cached_async_http_client.cache_clear()

    @functools.cache
    def _instrumented_cached_http_client(
        provider: str | None, timeout: int = DEFAULT_HTTP_TIMEOUT, connect: int = 5
    ) -> httpx.AsyncClient:
        async def response_hook(response: httpx.Response) -> None:
            await _log_response(case_logger, response)

        return httpx.AsyncClient(
            timeout=httpx.Timeout(timeout=timeout, connect=connect),
            headers={"User-Agent": get_user_agent()},
            event_hooks={"response": [response_hook]},
        )

    import pydantic_ai.models as pai_models  # noqa: PLC0415

    pai_models._cached_async_http_client = _instrumented_cached_http_client
    _INSTALLED = True
