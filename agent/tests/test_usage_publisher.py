import threading
from unittest.mock import MagicMock, patch

import pytest

from ai.usage_publisher import _Publisher, publish_agent_run_finished
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

    publisher = _Publisher()

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        msg = agents_pb2.AgentRunFinishedMessage(  # type: ignore[attr-defined]
            organization_id="org-123",
            chat_id="chat-456",
            model="claude-sonnet-4-6",
            input_tokens=100,
            output_tokens=300,
            total_tokens=500,
        )
        publisher.publish(msg.SerializeToString())

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

    publisher = _Publisher()

    with patch("ai.usage_publisher.pika") as mock_pika:
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection

        publisher.publish(b"first")
        publisher.publish(b"second")

    mock_pika.BlockingConnection.assert_called_once()
    assert mock_channel.basic_publish.call_count == 2


def test_publisher_reconnects_on_closed_connection() -> None:
    mock_channel = MagicMock()
    mock_channel.is_open = True

    mock_connection = MagicMock()
    mock_connection.is_open = False
    mock_connection.is_closed = True
    mock_connection.channel.return_value = mock_channel

    publisher = _Publisher()

    with patch("ai.usage_publisher.pika") as mock_pika:
        fresh_connection = MagicMock()
        fresh_connection.is_open = True
        fresh_connection.is_closed = False
        fresh_channel = MagicMock()
        fresh_channel.is_open = True
        fresh_connection.channel.return_value = fresh_channel

        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = [mock_connection, fresh_connection]

        publisher.publish(b"first")
        # First call creates mock_connection (is_open=False), so it reconnects
        publisher.publish(b"second")

    assert mock_pika.BlockingConnection.call_count >= 1
    assert fresh_channel.basic_publish.call_count >= 1


def test_publish_runs_in_background_thread() -> None:
    """Verify publish_agent_run_finished does not block the calling thread."""
    publish_thread_ids: list[int] = []
    caller_thread_id = threading.current_thread().ident

    def tracking_publish(self: _Publisher, body: bytes) -> None:
        publish_thread_ids.append(threading.current_thread().ident)  # type: ignore[arg-type]

    with (
        patch("ai.usage_publisher.config") as mock_config,
        patch.object(_Publisher, "publish", tracking_publish),
    ):
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"
        publish_agent_run_finished("org-1", "chat-1", "test", 10, 20, 100)

    # Wait for the background thread to finish
    import time

    time.sleep(0.1)

    assert len(publish_thread_ids) == 1
    assert publish_thread_ids[0] != caller_thread_id


def test_publish_skips_when_no_rabbitmq_url() -> None:
    with patch("ai.usage_publisher.config") as mock_config:
        mock_config.rabbitmq_url = ""

        publish_agent_run_finished("org-123", "chat-1", "test", 10, 20, 500)

    # No thread spawned, nothing to assert beyond no error


def test_publish_skips_when_total_tokens_zero_or_negative() -> None:
    threads_before = threading.active_count()

    with patch("ai.usage_publisher.config") as mock_config:
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"

        publish_agent_run_finished("org-123", "chat-1", "test", 0, 0, 0)
        publish_agent_run_finished("org-123", "chat-1", "test", 0, 0, -10)

    # No background threads should have been spawned
    assert threading.active_count() <= threads_before


def test_publish_fails_silently_on_connection_error(capsys: pytest.CaptureFixture[str]) -> None:
    import time

    with (
        patch("ai.usage_publisher.config") as mock_config,
        patch("ai.usage_publisher.pika") as mock_pika,
    ):
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = ConnectionError("refused")

        # Reset the global publisher so it tries to connect fresh
        with patch("ai.usage_publisher._publisher", _Publisher()):
            publish_agent_run_finished("org-123", "chat-1", "test", 10, 20, 500)
            time.sleep(0.2)

    captured = capsys.readouterr()
    assert "failed to publish agent run finished" in captured.out
