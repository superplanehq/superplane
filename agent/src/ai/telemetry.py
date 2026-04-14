from __future__ import annotations

import logging

from ai.config import config

log = logging.getLogger(__name__)


def init_telemetry() -> None:
    if not config.otel_enabled:
        return

    if getattr(init_telemetry, "_done", False):
        return

    try:
        import logfire

        logfire.configure(send_to_logfire=False)
        logfire.instrument_pydantic_ai()
        init_telemetry._done = True  # type: ignore[attr-defined]
    except Exception:
        log.warning("Failed to initialize telemetry", exc_info=True)
        return

    log.info("Pydantic AI telemetry initialized (logfire -> OTEL collector)")


def shutdown_telemetry() -> None:
    pass
