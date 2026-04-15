from __future__ import annotations

import logging

from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.metrics import set_meter_provider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import set_tracer_provider

from ai.config import config

log = logging.getLogger(__name__)

_tracer_provider: TracerProvider | None = None
_meter_provider: MeterProvider | None = None


def init_telemetry() -> None:
    global _tracer_provider, _meter_provider  # noqa: PLW0603

    if not config.otel_enabled:
        return

    if _tracer_provider is not None:
        return

    try:
        from pydantic_ai import Agent

        _tracer_provider = TracerProvider()
        _tracer_provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
        set_tracer_provider(_tracer_provider)

        _meter_provider = MeterProvider(
            metric_readers=[PeriodicExportingMetricReader(OTLPMetricExporter())]
        )
        set_meter_provider(_meter_provider)

        Agent.instrument_all()
    except Exception:
        log.warning("Failed to initialize telemetry", exc_info=True)
        _tracer_provider = None
        _meter_provider = None
        return

    log.info("OpenTelemetry initialized (gRPC OTLP exporter, pydantic-ai instrumented)")


def shutdown_telemetry() -> None:
    if _meter_provider is not None:
        _meter_provider.shutdown()
    if _tracer_provider is not None:
        _tracer_provider.shutdown()


def init_sentry() -> None:
    if not config.sentry_dsn:
        return

    try:
        import sentry_sdk

        sentry_sdk.init(
            dsn=config.sentry_dsn,
            environment=config.sentry_environment or None,
            enable_tracing=False,
        )
    except Exception:
        log.warning("Failed to initialize Sentry", exc_info=True)
        return

    log.info("Sentry telemetry initialized")


def shutdown_sentry() -> None:
    if not config.sentry_dsn:
        return

    try:
        import sentry_sdk

        sentry_sdk.flush(timeout=2)
    except Exception:
        log.warning("Failed to flush Sentry", exc_info=True)
