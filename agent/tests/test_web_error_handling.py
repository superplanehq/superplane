import asyncio
from collections.abc import AsyncIterator
from typing import Any
from unittest.mock import AsyncMock, patch

import pytest

from ai.web import (
    _extract_status_code,
    _friendly_error_message,
    _is_transient_error,
    _run_stream_events,
)


class FakeAPIStatusError(Exception):
    def __init__(self, status_code: int) -> None:
        super().__init__(f"API error {status_code}")
        self.status_code = status_code


class FakeWrappedError(Exception):
    """Simulates pydantic-ai wrapping an underlying API error as __cause__."""


# --- _extract_status_code ---


class TestExtractStatusCode:
    def test_direct_status_code(self) -> None:
        assert _extract_status_code(FakeAPIStatusError(529)) == 529

    def test_chained_cause(self) -> None:
        inner = FakeAPIStatusError(429)
        outer = FakeWrappedError("wrapped")
        outer.__cause__ = inner
        assert _extract_status_code(outer) == 429

    def test_chained_context(self) -> None:
        inner = FakeAPIStatusError(503)
        outer = FakeWrappedError("wrapped")
        outer.__context__ = inner
        assert _extract_status_code(outer) == 503

    def test_no_status_code(self) -> None:
        assert _extract_status_code(ValueError("plain error")) is None

    def test_non_int_status_code_ignored(self) -> None:
        error = Exception("bad")
        error.status_code = "not-an-int"  # type: ignore[attr-defined]
        assert _extract_status_code(error) is None


# --- _is_transient_error ---


class TestIsTransientError:
    @pytest.mark.parametrize("code", [429, 502, 503, 504, 529])
    def test_transient_status_codes(self, code: int) -> None:
        assert _is_transient_error(FakeAPIStatusError(code)) is True

    def test_non_transient_status_code(self) -> None:
        assert _is_transient_error(FakeAPIStatusError(400)) is False
        assert _is_transient_error(FakeAPIStatusError(401)) is False
        assert _is_transient_error(FakeAPIStatusError(500)) is False

    def test_no_status_code(self) -> None:
        assert _is_transient_error(ValueError("plain")) is False

    def test_chained_transient_error(self) -> None:
        inner = FakeAPIStatusError(529)
        outer = FakeWrappedError("wrapped")
        outer.__cause__ = inner
        assert _is_transient_error(outer) is True


# --- _friendly_error_message ---


class TestFriendlyErrorMessage:
    def test_overloaded_529(self) -> None:
        msg = _friendly_error_message(FakeAPIStatusError(529))
        assert "overloaded" in msg.lower()
        assert "try again" in msg.lower()

    def test_rate_limit_429(self) -> None:
        msg = _friendly_error_message(FakeAPIStatusError(429))
        assert "rate limit" in msg.lower()

    @pytest.mark.parametrize("code", [502, 503, 504])
    def test_gateway_errors(self, code: int) -> None:
        msg = _friendly_error_message(FakeAPIStatusError(code))
        assert "unavailable" in msg.lower()

    def test_client_error_4xx(self) -> None:
        msg = _friendly_error_message(FakeAPIStatusError(403))
        assert "configuration" in msg.lower()

    def test_unknown_error(self) -> None:
        msg = _friendly_error_message(ValueError("something broke"))
        assert "unexpected" in msg.lower()

    def test_chained_error_extracts_status(self) -> None:
        inner = FakeAPIStatusError(529)
        outer = FakeWrappedError("wrapped")
        outer.__cause__ = inner
        msg = _friendly_error_message(outer)
        assert "overloaded" in msg.lower()


# --- _run_stream_events retry behavior ---


class TestRunStreamEventsRetry:
    def _make_agent(self, side_effects: list[Any]) -> Any:
        """Build a mock agent whose run_stream_events follows the given side effects.

        Each element is either a list of events to yield, or an Exception to raise.
        """
        call_index = 0

        async def fake_run_stream_events(**kwargs: Any) -> AsyncIterator[Any]:
            nonlocal call_index
            effect = side_effects[call_index]
            call_index += 1
            if isinstance(effect, Exception):
                raise effect
            for item in effect:
                yield item

        agent = AsyncMock()
        agent.run_stream_events = fake_run_stream_events
        return agent, lambda: call_index

    def test_success_on_first_attempt(self) -> None:
        agent, call_count = self._make_agent([["event1", "event2"]])
        events: list[Any] = []

        async def run() -> None:
            async for event in _run_stream_events(agent):
                events.append(event)

        asyncio.run(run())
        assert events == ["event1", "event2"]
        assert call_count() == 1

    @patch("ai.web.asyncio.sleep", new_callable=AsyncMock)
    def test_retries_on_transient_error(self, mock_sleep: AsyncMock) -> None:
        agent, call_count = self._make_agent(
            [
                FakeAPIStatusError(529),
                ["event1"],
            ]
        )
        events: list[Any] = []

        async def run() -> None:
            async for event in _run_stream_events(agent):
                events.append(event)

        asyncio.run(run())
        assert events == ["event1"]
        assert call_count() == 2
        mock_sleep.assert_called_once()

    def test_no_retry_on_non_transient_error(self) -> None:
        agent, call_count = self._make_agent([FakeAPIStatusError(400)])

        async def run() -> list[Any]:
            events: list[Any] = []
            async for event in _run_stream_events(agent):
                events.append(event)
            return events

        with pytest.raises(FakeAPIStatusError):
            asyncio.run(run())
        assert call_count() == 1

    @patch("ai.web.asyncio.sleep", new_callable=AsyncMock)
    def test_raises_after_max_retries(self, mock_sleep: AsyncMock) -> None:
        agent, call_count = self._make_agent(
            [
                FakeAPIStatusError(529),
                FakeAPIStatusError(529),
                FakeAPIStatusError(529),
            ]
        )

        async def run() -> list[Any]:
            events: list[Any] = []
            async for event in _run_stream_events(agent):
                events.append(event)
            return events

        with pytest.raises(FakeAPIStatusError):
            asyncio.run(run())
        assert call_count() == 3

    def test_no_retry_after_yielding_events(self) -> None:
        """Once events have been yielded, mid-stream errors must not retry."""
        call_index = 0

        async def failing_mid_stream(**kwargs: Any) -> AsyncIterator[str]:
            nonlocal call_index
            call_index += 1
            yield "event1"
            raise FakeAPIStatusError(529)

        agent = AsyncMock()
        agent.run_stream_events = failing_mid_stream

        events: list[Any] = []

        async def run() -> None:
            async for event in _run_stream_events(agent):
                events.append(event)

        with pytest.raises(FakeAPIStatusError):
            asyncio.run(run())
        assert events == ["event1"]
        assert call_index == 1
