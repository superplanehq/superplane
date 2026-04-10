"""OpenTelemetry bridge: send pydantic-ai spans to eval case log files.

Enabled for eval runs when :func:`build_eval_instrumentation` returns non-``None``
(see ``EVAL_OTEL_CASE_LOG`` in :mod:`evals.runner`).

Uses a :class:`opentelemetry.sdk.trace.export.SimpleSpanProcessor` so spans are
exported on the same thread as span end (preserves asyncio ``ContextVar`` for
the active case name).
"""

from __future__ import annotations

import os
from collections.abc import Sequence
from contextvars import ContextVar
from typing import Any

from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import ReadableSpan, TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor, SpanExporter, SpanExportResult
from pydantic_ai.models.instrumented import InstrumentationSettings

from evals.case_logger import CaseLogger

# Set in :mod:`evals.case_task` around each agent run.
current_eval_case_name: ContextVar[str | None] = ContextVar("current_eval_case_name", default=None)

# Attributes that duplicate huge JSON already visible elsewhere in the log.
_SKIP_ATTR_KEYS = frozenset(
    {
        "gen_ai.input.messages",
        "gen_ai.output.messages",
        "gen_ai.system_instructions",
        "pydantic_ai.all_messages",
        "all_messages_events",
        "logfire.json_schema",
        "metadata",
    }
)


def _llm_full_span_log_enabled() -> bool:
    """Default: full ``span.to_json()`` per span. ``EVAL_LLM_FULL_LOG=0``: one-line compact OTEL."""

    value = os.getenv("EVAL_LLM_FULL_LOG", "1").strip().lower()
    return value not in ("0", "false", "no")


def _truncate(value: Any, max_len: int) -> Any:
    if isinstance(value, str) and len(value) > max_len:
        return f"{value[:max_len]}…(len={len(value)})"
    return value


def _format_attributes(attrs: Any, *, compact: bool) -> dict[str, Any]:
    """Used only for compact one-line OTEL (``EVAL_LLM_FULL_LOG=0``)."""

    if not attrs:
        return {}
    out: dict[str, Any] = {}
    for key, raw in attrs.items():
        if compact and key in _SKIP_ATTR_KEYS:
            continue
        if compact and isinstance(key, str) and key.startswith("gen_ai.tool.definitions"):
            continue
        max_len = 120 if compact else 10_000_000
        val = _truncate(raw, max_len)
        out[str(key)] = val
    return out


def _span_line(span: ReadableSpan) -> str:
    name = span.name
    start = span.start_time or 0
    end = span.end_time or start
    duration_ms = (end - start) / 1_000_000.0
    attrs = _format_attributes(span.attributes, compact=True)
    parts = [f'OTEL_SPAN name={name!r} duration_ms={duration_ms:.3f}']
    for key in sorted(attrs):
        parts.append(f"{key}={attrs[key]!r}")
    return " ".join(parts)


class CaseLogSpanExporter(SpanExporter):
    """Writes span records to the active eval case log (see context var)."""

    def __init__(self, case_logger: CaseLogger) -> None:
        self._case_logger = case_logger

    def export(self, spans: Sequence[ReadableSpan]) -> SpanExportResult:
        case_name = current_eval_case_name.get()
        if case_name is None:
            return SpanExportResult.SUCCESS
        for span in spans:
            if _llm_full_span_log_enabled():
                self._case_logger.log_case_sync(
                    case_name,
                    "OTEL_SPAN_JSON\n" + span.to_json(indent=2),
                )
            else:
                self._case_logger.log_case_sync(case_name, _span_line(span))
        return SpanExportResult.SUCCESS

    def shutdown(self) -> None:
        return None

    def force_flush(self, timeout_millis: int = 30000) -> bool:
        return True


def build_eval_instrumentation(case_logger: CaseLogger) -> InstrumentationSettings:
    """Build pydantic-ai instrumentation that records spans into the eval case log."""

    resource = Resource.create({"service.name": "superplane-agent-eval"})
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(SimpleSpanProcessor(CaseLogSpanExporter(case_logger)))
    return InstrumentationSettings(
        tracer_provider=provider,
        include_content=True,
        version=4,
    )
