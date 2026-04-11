import threading
import time
from unittest.mock import MagicMock, patch

import pytest

from ai.usage_publisher import NoopUsagePublisher, UsagePublisher, _Connection
from private import agents_pb2


def test_publisher_serializes_protobuf_correctly() -> None:
    """Verify the published body can be deserialized back into the proto message."""
    published_bodies: list[bytes] = []
    mock_channel = MagicMock()
    mock_channel.is_open = True
    mock_connection = MagicMock()
    mock_connection.is_open = True
    mock_connection.is_closed = False
    mock_connection.channel.return_value = mock_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        def capture_publish(**kwargs: object) -> None:
            published_bodies.append(kwargs["body"])  # type: ignore[arg-type]

        mock_channel.basic_publish.side_effect = capture_publish

        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished(
            "org-123", "chat-456", "claude-sonnet-4-6", 100, 300, 500, 50, 25
        )
        time.sleep(0.3)
        publisher.close()

    assert len(published_bodies) == 1
    parsed = agents_pb2.AgentRunFinishedMessage()  # type: ignore[attr-defined]
    parsed.ParseFromString(published_bodies[0])
    assert parsed.organization_id == "org-123"
    assert parsed.chat_id == "chat-456"
    assert parsed.model == "claude-sonnet-4-6"
    assert parsed.input_tokens == 100
    assert parsed.output_tokens == 300
    assert parsed.total_tokens == 500
    assert parsed.cache_read_tokens == 50
    assert parsed.cache_write_tokens == 25


def test_publisher_reuses_connection() -> None:
    mock_channel = MagicMock()
    mock_channel.is_open = True
    mock_connection = MagicMock()
    mock_connection.is_open = True
    mock_connection.is_closed = False
    mock_connection.channel.return_value = mock_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        conn = _Connection()
        conn.ensure("amqp://localhost")
        conn.ensure("amqp://localhost")

    mock_pika.BlockingConnection.assert_called_once()


def test_connection_reconnects_when_closed() -> None:
    closed_conn = MagicMock()
    closed_conn.is_open = False
    closed_conn.is_closed = True

    fresh_channel = MagicMock()
    fresh_channel.is_open = True
    fresh_conn = MagicMock()
    fresh_conn.is_open = True
    fresh_conn.is_closed = False
    fresh_conn.channel.return_value = fresh_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = [closed_conn, fresh_conn]

        conn = _Connection()
        conn.ensure("amqp://localhost")
        conn.ensure("amqp://localhost")

    assert mock_pika.BlockingConnection.call_count == 2


def test_publish_does_not_block_caller() -> None:
    publish_thread_ids: list[int] = []
    caller_thread_id = threading.current_thread().ident

    def tracking_publish(self: UsagePublisher, conn: _Connection, body: bytes) -> None:
        publish_thread_ids.append(threading.current_thread().ident)  # type: ignore[arg-type]

    with patch.object(UsagePublisher, "_publish", tracking_publish):
        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished("org-1", "c-1", "test", 10, 20, 100, 5, 3)
        time.sleep(0.3)
        publisher.close()

    assert len(publish_thread_ids) == 1
    assert publish_thread_ids[0] != caller_thread_id


def test_publish_skips_when_total_tokens_zero_or_negative() -> None:
    with patch("ai.usage_publisher.pika"):
        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished("org-123", "c-1", "test", 0, 0, 0, 0, 0)
        publisher.publish_agent_run_finished("org-123", "c-1", "test", 0, 0, -10, 0, 0)
        assert publisher._queue.empty()
        publisher.close()


def test_publish_fails_silently_on_connection_error(capsys: pytest.CaptureFixture[str]) -> None:
    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = ConnectionError("refused")

        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished("org-123", "c-1", "test", 10, 20, 500, 5, 3)
        time.sleep(0.3)
        publisher.close()

    captured = capsys.readouterr()
    assert "failed to publish agent run finished" in captured.out


def test_noop_publisher_does_nothing() -> None:
    publisher = NoopUsagePublisher()
    publisher.publish_agent_run_finished("org-1", "c-1", "test", 10, 20, 100, 5, 3)
    publisher.close()


def test_close_shuts_down_publisher_thread() -> None:
    with patch("ai.usage_publisher.pika"):
        publisher = UsagePublisher("amqp://localhost")
        assert publisher._thread.is_alive()
        publisher.close()
        assert not publisher._thread.is_alive()


def test_connection_close_cleans_up() -> None:
    mock_connection = MagicMock()
    mock_connection.is_open = True
    mock_connection.is_closed = False
    mock_channel = MagicMock()
    mock_channel.is_open = True
    mock_connection.channel.return_value = mock_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        conn = _Connection()
        conn.ensure("amqp://localhost")
        assert conn.is_open
        conn.close()

    mock_connection.close.assert_called_once()
    assert not conn.is_open
