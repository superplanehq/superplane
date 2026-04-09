from __future__ import annotations

import logging
import os
import threading
from typing import TYPE_CHECKING

from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader

if TYPE_CHECKING:
    from pydantic_ai.usage import RunUsage

log = logging.getLogger(__name__)

_lock = threading.Lock()
_meter_provider: MeterProvider | None = None
_agent_run_tokens: metrics.Histogram | None = None
_agent_run_duration: metrics.Histogram | None = None
_agent_run_tool_calls: metrics.Histogram | None = None
_agent_run_model_requests: metrics.Histogram | None = None
_agent_run_outcome: metrics.Counter | None = None
_agent_run_first_token_duration: metrics.Histogram | None = None
_agent_tool_duration: metrics.Histogram | None = None
_agent_run_retries: metrics.Histogram | None = None


def init_metrics() -> None:
    global _meter_provider, _agent_run_tokens  # noqa: PLW0603
    global _agent_run_duration, _agent_run_tool_calls, _agent_run_model_requests  # noqa: PLW0603
    global _agent_run_outcome, _agent_run_first_token_duration  # noqa: PLW0603
    global _agent_tool_duration, _agent_run_retries  # noqa: PLW0603

    if os.getenv("OTEL_ENABLED") != "yes":
        return

    with _lock:
        if _meter_provider is not None:
            return

        try:
            exporter = OTLPMetricExporter()
            reader = PeriodicExportingMetricReader(exporter)
            _meter_provider = MeterProvider(metric_readers=[reader])
            metrics.set_meter_provider(_meter_provider)

            meter = _meter_provider.get_meter("superplane-agent")
            _agent_run_tokens = meter.create_histogram(
                name="agent.run.tokens",
                description="Token count per agent run",
                unit="1",
            )
            _agent_run_duration = meter.create_histogram(
                name="agent.run.duration",
                description="End-to-end wall-clock time of an agent run",
                unit="s",
            )
            _agent_run_tool_calls = meter.create_histogram(
                name="agent.run.tool_calls",
                description="Number of tool calls executed per agent run",
                unit="1",
            )
            _agent_run_model_requests = meter.create_histogram(
                name="agent.run.model_requests",
                description="Number of LLM API requests per agent run",
                unit="1",
            )
            _agent_run_outcome = meter.create_counter(
                name="agent.run.outcome",
                description="Agent run count by outcome",
                unit="1",
            )
            _agent_run_first_token_duration = meter.create_histogram(
                name="agent.run.first_token.duration",
                description="Time from run start to first streamed token",
                unit="s",
            )
            _agent_tool_duration = meter.create_histogram(
                name="agent.tool.duration",
                description="Execution time of each individual tool call",
                unit="s",
            )
            _agent_run_retries = meter.create_histogram(
                name="agent.run.retries",
                description="Estimated LLM retry count per agent run",
                unit="1",
            )
        except Exception:
            log.warning("Failed to initialize OpenTelemetry metrics", exc_info=True)
            _meter_provider = None
            _agent_run_tokens = None
            _agent_run_duration = None
            _agent_run_tool_calls = None
            _agent_run_model_requests = None
            _agent_run_outcome = None
            _agent_run_first_token_duration = None
            _agent_tool_duration = None
            _agent_run_retries = None
            return

    log.info("OpenTelemetry metrics initialized")


def shutdown_metrics() -> None:
    global _meter_provider  # noqa: PLW0603
    with _lock:
        if _meter_provider is not None:
            _meter_provider.shutdown()
            _meter_provider = None


def record_agent_run_tokens(usage: RunUsage) -> None:
    histogram = _agent_run_tokens
    if histogram is None:
        return

    if usage.input_tokens > 0:
        histogram.record(usage.input_tokens, {"token_type": "input"})
    if usage.output_tokens > 0:
        histogram.record(usage.output_tokens, {"token_type": "output"})
    if usage.cache_read_tokens > 0:
        histogram.record(usage.cache_read_tokens, {"token_type": "cache_read"})
    if usage.cache_write_tokens > 0:
        histogram.record(usage.cache_write_tokens, {"token_type": "cache_write"})


def record_agent_run_duration(seconds: float) -> None:
    histogram = _agent_run_duration
    if histogram is not None:
        histogram.record(seconds)


def record_agent_run_tool_calls(count: int) -> None:
    histogram = _agent_run_tool_calls
    if histogram is not None:
        histogram.record(count)


def record_agent_run_model_requests(count: int) -> None:
    histogram = _agent_run_model_requests
    if histogram is not None:
        histogram.record(count)


def record_agent_run_outcome(outcome: str) -> None:
    counter = _agent_run_outcome
    if counter is not None:
        counter.add(1, {"outcome": outcome})


def record_agent_run_first_token_duration(seconds: float) -> None:
    histogram = _agent_run_first_token_duration
    if histogram is not None:
        histogram.record(seconds)


def record_agent_tool_duration(tool_name: str, seconds: float) -> None:
    histogram = _agent_tool_duration
    if histogram is not None:
        histogram.record(seconds, {"tool_name": tool_name})


def record_agent_run_retries(count: int) -> None:
    histogram = _agent_run_retries
    if histogram is not None:
        histogram.record(count)
