"""Publishes agent run finished events to RabbitMQ for the saas usage service."""

from __future__ import annotations

import pika  # type: ignore[import-untyped]
import pika.exceptions  # type: ignore[import-untyped]

from ai.config import config
from private import agents_pb2

AGENT_EXCHANGE = "superplane.agent-exchange"
AGENT_RUN_FINISHED_ROUTING_KEY = "agent-run-finished"


def publish_agent_run_finished(
    organization_id: str,
    chat_id: str,
    model: str,
    input_tokens: int,
    output_tokens: int,
    total_tokens: int,
) -> None:
    """Publish an agent-run-finished message to RabbitMQ.

    Fails silently with a log message if RabbitMQ is unavailable.
    """
    rabbitmq_url = config.rabbitmq_url
    if not rabbitmq_url:
        return

    if total_tokens <= 0:
        return

    connection = None
    try:
        message = agents_pb2.AgentRunFinishedMessage(  # type: ignore[attr-defined]
            organization_id=organization_id,
            chat_id=chat_id,
            model=model,
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            total_tokens=total_tokens,
        )
        body = message.SerializeToString()

        params = pika.URLParameters(rabbitmq_url)
        connection = pika.BlockingConnection(params)
        channel = connection.channel()
        channel.exchange_declare(exchange=AGENT_EXCHANGE, exchange_type="topic", durable=True)
        channel.basic_publish(
            exchange=AGENT_EXCHANGE,
            routing_key=AGENT_RUN_FINISHED_ROUTING_KEY,
            body=body,
        )
    except Exception as error:
        print(f"[web] failed to publish agent run finished: {error}", flush=True)
    finally:
        if connection and not connection.is_closed:
            connection.close()
