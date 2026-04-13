from __future__ import annotations

import logging
import os

log = logging.getLogger(__name__)

_initialized = False


def init_telemetry() -> None:
    global _initialized  # noqa: PLW0603

    if os.getenv("OTEL_ENABLED") != "yes":
        return

    if _initialized:
        return

    try:
        import logfire

        logfire.configure(send_to_logfire=False)
        logfire.instrument_pydantic_ai()
        _initialized = True
    except Exception:
        log.warning("Failed to initialize telemetry", exc_info=True)
        return

    log.info("Pydantic AI telemetry initialized (logfire -> OTEL collector)")


def shutdown_telemetry() -> None:
    pass
