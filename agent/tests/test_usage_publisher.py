from unittest.mock import MagicMock, patch

import pytest

from ai.usage_publisher import publish_agent_tokens_used
from private import agents_pb2


def test_publish_serializes_protobuf_correctly() -> None:
    """Verify the published body can be deserialized back into the proto message."""
    published_bodies: list[bytes] = []

    with patch("ai.usage_publisher.config") as mock_config, patch(
        "ai.usage_publisher.pika"
    ) as mock_pika:
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"
        mock_connection = MagicMock()
        mock_channel = MagicMock()
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection
        mock_connection.channel.return_value = mock_channel

        def capture_publish(**kwargs: object) -> None:
            published_bodies.append(kwargs["body"])  # type: ignore[arg-type]

        mock_channel.basic_publish.side_effect = capture_publish

        publish_agent_tokens_used("org-123", 500)

    assert len(published_bodies) == 1
    parsed = agents_pb2.AgentTokensUsedMessage()
    parsed.ParseFromString(published_bodies[0])
    assert parsed.organization_id == "org-123"
    assert parsed.tokens == 500


def test_publish_uses_correct_exchange_and_routing_key() -> None:
    with patch("ai.usage_publisher.config") as mock_config, patch(
        "ai.usage_publisher.pika"
    ) as mock_pika:
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"
        mock_connection = MagicMock()
        mock_channel = MagicMock()
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.return_value = mock_connection
        mock_connection.channel.return_value = mock_channel

        publish_agent_tokens_used("org-456", 100)

    mock_channel.exchange_declare.assert_called_once_with(
        exchange="superplane.agent-exchange", exchange_type="topic", durable=True
    )
    mock_channel.basic_publish.assert_called_once()
    call_kwargs = mock_channel.basic_publish.call_args[1]
    assert call_kwargs["exchange"] == "superplane.agent-exchange"
    assert call_kwargs["routing_key"] == "agent-tokens-used"


def test_publish_skips_when_no_rabbitmq_url() -> None:
    with patch("ai.usage_publisher.config") as mock_config, patch(
        "ai.usage_publisher.pika"
    ) as mock_pika:
        mock_config.rabbitmq_url = ""

        publish_agent_tokens_used("org-123", 500)

    mock_pika.BlockingConnection.assert_not_called()


def test_publish_skips_when_tokens_zero_or_negative() -> None:
    with patch("ai.usage_publisher.config") as mock_config, patch(
        "ai.usage_publisher.pika"
    ) as mock_pika:
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"

        publish_agent_tokens_used("org-123", 0)
        publish_agent_tokens_used("org-123", -10)

    mock_pika.BlockingConnection.assert_not_called()


def test_publish_fails_silently_on_connection_error(capsys: pytest.CaptureFixture[str]) -> None:
    with patch("ai.usage_publisher.config") as mock_config, patch(
        "ai.usage_publisher.pika"
    ) as mock_pika:
        mock_config.rabbitmq_url = "amqp://guest:guest@localhost:5672"
        mock_pika.URLParameters.return_value = "params"
        mock_pika.BlockingConnection.side_effect = ConnectionError("refused")

        publish_agent_tokens_used("org-123", 500)

    captured = capsys.readouterr()
    assert "failed to publish agent token usage" in captured.out
