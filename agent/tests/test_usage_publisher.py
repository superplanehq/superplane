import threading
import time
from unittest.mock import MagicMock, patch

import pytest

from ai.usage_publisher import NoopUsagePublisher, UsagePublisher
from private import agents_pb2


def test_publisher_serializes_protobuf_correctly() -> None:
    """Verify the published body can be deserialized back into the proto message."""
    published_bodies: list[bytes] = []
    mock_channel = MagicMock()
    mock_channel.is_open = True

    def capture_publish(**kwargs: object) -> None:
        published_bodies.append(kwargs["body"])  # type: ignore[arg-type]

    mock_channel.basic_publish.side_effect = capture_publish

    mock_connection = MagicMock()
    mock_connection.is_open = True
    mock_connection.is_closed = False
    mock_connection.channel.return_value = mock_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished(
            "org-123", "chat-456", "claude-sonnet-4-6", 100, 300, 500
        )
        time.sleep(0.1)

    assert len(published_bodies) == 1
    parsed = agents_pb2.AgentRunFinishedMessage()  # type: ignore[attr-defined]
    parsed.ParseFromString(published_bodies[0])
    assert parsed.organization_id == "org-123"
    assert parsed.chat_id == "chat-456"
    assert parsed.model == "claude-sonnet-4-6"
    assert parsed.input_tokens == 100
    assert parsed.output_tokens == 300
    assert parsed.total_tokens == 500


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

        publisher = UsagePublisher("amqp://localhost")
        # Call _publish directly to avoid threading for this test
        publisher._publish(b"first")
        publisher._publish(b"second")

    mock_pika.BlockingConnection.assert_called_once()
    assert mock_channel.basic_publish.call_count == 2


def test_publisher_reconnects_on_closed_connection() -> None:
    closed_connection = MagicMock()
    closed_connection.is_open = False
    closed_connection.is_closed = True

    fresh_channel = MagicMock()
    fresh_channel.is_open = True
    fresh_connection = MagicMock()
    fresh_connection.is_open = True
    fresh_connection.is_closed = False
    fresh_connection.channel.return_value = fresh_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = [closed_connection, fresh_connection]

        publisher = UsagePublisher("amqp://localhost")
        publisher._publish(b"first")
        publisher._publish(b"second")

    assert mock_pika.BlockingConnection.call_count == 2
    assert fresh_channel.basic_publish.call_count >= 1


def test_publish_runs_in_background_thread() -> None:
    publish_thread_ids: list[int] = []
    caller_thread_id = threading.current_thread().ident

    def tracking_publish(self: UsagePublisher, body: bytes) -> None:
        publish_thread_ids.append(threading.current_thread().ident)  # type: ignore[arg-type]

    with patch.object(UsagePublisher, "_publish", tracking_publish):
        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished("org-1", "chat-1", "test", 10, 20, 100)

    time.sleep(0.1)

    assert len(publish_thread_ids) == 1
    assert publish_thread_ids[0] != caller_thread_id


def test_publish_skips_when_total_tokens_zero_or_negative() -> None:
    publisher = UsagePublisher("amqp://localhost")

    with patch.object(publisher, "_publish") as mock_publish:
        publisher.publish_agent_run_finished("org-123", "chat-1", "test", 0, 0, 0)
        publisher.publish_agent_run_finished("org-123", "chat-1", "test", 0, 0, -10)

    mock_publish.assert_not_called()


def test_publish_fails_silently_on_connection_error(capsys: pytest.CaptureFixture[str]) -> None:
    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = ConnectionError("refused")

        publisher = UsagePublisher("amqp://localhost")
        publisher.publish_agent_run_finished("org-123", "chat-1", "test", 10, 20, 500)
        time.sleep(0.2)

    captured = capsys.readouterr()
    assert "failed to publish agent run finished" in captured.out


def test_noop_publisher_does_nothing() -> None:
    publisher = NoopUsagePublisher()
    publisher.publish_agent_run_finished("org-1", "chat-1", "test", 10, 20, 100)
    publisher.close()


def test_close_cleans_up_connection() -> None:
    mock_connection = MagicMock()
    mock_connection.is_open = True
    mock_connection.is_closed = False
    mock_channel = MagicMock()
    mock_channel.is_open = True
    mock_connection.channel.return_value = mock_channel

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        publisher = UsagePublisher("amqp://localhost")
        publisher._publish(b"hello")
        publisher.close()

    mock_connection.close.assert_called_once()
