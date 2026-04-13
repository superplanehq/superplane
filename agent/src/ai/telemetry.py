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


def init_metrics() -> None:
    global _meter_provider, _agent_run_tokens  # noqa: PLW0603

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
        except Exception:
            log.warning("Failed to initialize OpenTelemetry metrics", exc_info=True)
            _meter_provider = None
            _agent_run_tokens = None
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
