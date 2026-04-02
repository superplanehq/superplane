import asyncio
import os
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

_DEFAULT_DRAIN_TIMEOUT = 300.0
_DRAIN_LOG_INTERVAL = 5.0


def _resolve_drain_timeout() -> float:
    raw = os.getenv("DRAIN_TIMEOUT", "").strip()
    if not raw:
        return _DEFAULT_DRAIN_TIMEOUT
    try:
        return max(float(raw), 0.0)
    except ValueError:
        return _DEFAULT_DRAIN_TIMEOUT


class ActiveStreamTracker:
    """Tracks in-flight SSE streams so the service can drain them before shutting down."""

    def __init__(self, drain_timeout: float | None = None) -> None:
        self._active_count = 0
        self._lock = asyncio.Lock()
        self._drained = asyncio.Event()
        self._drained.set()
        self._shutting_down = False
        self._drain_timeout = (
            drain_timeout if drain_timeout is not None else _resolve_drain_timeout()
        )

    @property
    def is_shutting_down(self) -> bool:
        return self._shutting_down

    @property
    def active_count(self) -> int:
        return self._active_count

    async def acquire(self) -> None:
        async with self._lock:
            self._active_count += 1
            self._drained.clear()

    async def release(self) -> None:
        async with self._lock:
            self._active_count -= 1
            if self._active_count == 0:
                self._drained.set()

    @asynccontextmanager
    async def track_stream(self) -> AsyncIterator[None]:
        await self.acquire()
        try:
            yield
        finally:
            await self.release()

    def begin_shutdown(self) -> None:
        self._shutting_down = True

    async def wait_for_drain(self) -> None:
        if self._active_count == 0:
            print("[web] graceful shutdown: no active streams, proceeding with cleanup", flush=True)
            return

        print(
            f"[web] graceful shutdown: waiting for {self._active_count} active stream(s) to finish"
            f" (timeout={self._drain_timeout}s)...",
            flush=True,
        )
        try:
            await asyncio.wait_for(self._drain_with_logging(), timeout=self._drain_timeout)
            print(
                "[web] graceful shutdown: all streams finished, proceeding with cleanup",
                flush=True,
            )
        except TimeoutError:
            remaining = self._active_count
            print(
                f"[web] graceful shutdown: drain timeout reached"
                f" with {remaining} stream(s) still active,"
                " forcing shutdown",
                flush=True,
            )

    async def _drain_with_logging(self) -> None:
        while not self._drained.is_set():
            try:
                await asyncio.wait_for(self._drained.wait(), timeout=_DRAIN_LOG_INTERVAL)
            except TimeoutError:
                remaining = self._active_count
                print(
                    f"[web] graceful shutdown: waiting for"
                    f" {remaining} active stream(s) to finish...",
                    flush=True,
                )
