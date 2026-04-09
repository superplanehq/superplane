"""Publishes agent token usage events to RabbitMQ for the saas usage service."""

from __future__ import annotations

import pika
import pika.exceptions

from ai.config import config
from private import agents_pb2

AGENT_EXCHANGE = "superplane.agent-exchange"
AGENT_TOKENS_USED_ROUTING_KEY = "agent-tokens-used"


def publish_agent_tokens_used(organization_id: str, tokens: int) -> None:
    """Publish an agent-tokens-used message to RabbitMQ.

    Fails silently with a log message if RabbitMQ is unavailable.
    """
    rabbitmq_url = config.rabbitmq_url
    if not rabbitmq_url:
        return

    if tokens <= 0:
        return

    try:
        message = agents_pb2.AgentTokensUsedMessage(
            organization_id=organization_id,
            tokens=tokens,
        )
        body = message.SerializeToString()

        params = pika.URLParameters(rabbitmq_url)
        connection = pika.BlockingConnection(params)
        channel = connection.channel()
        channel.exchange_declare(exchange=AGENT_EXCHANGE, exchange_type="topic", durable=True)
        channel.basic_publish(
            exchange=AGENT_EXCHANGE,
            routing_key=AGENT_TOKENS_USED_ROUTING_KEY,
            body=body,
        )
        connection.close()
    except Exception as error:
        print(f"[web] failed to publish agent token usage: {error}", flush=True)
