from __future__ import annotations

import logging
import os

from ai.config import config

log = logging.getLogger(__name__)


def init_telemetry() -> None:
    if not config.otel_enabled:
        return

    if getattr(init_telemetry, "_done", False):
        return

    try:
        import logfire

        os.environ.setdefault("OTEL_LOGS_EXPORTER", "none")
        logfire.configure(send_to_logfire=False)
        logfire.instrument_pydantic_ai()
        init_telemetry._done = True  # type: ignore[attr-defined]
    except Exception:
        log.warning("Failed to initialize telemetry", exc_info=True)
        return

    log.info("Pydantic AI telemetry initialized (logfire -> OTEL collector)")


def shutdown_telemetry() -> None:
    pass


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
