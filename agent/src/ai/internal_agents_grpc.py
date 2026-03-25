import os
import threading
from concurrent import futures
from dataclasses import dataclass

import grpc
from google.protobuf.timestamp_pb2 import Timestamp

from ai.session_store import AgentNotFoundError, SessionStore, StoredAgent, StoredAgentMessage
from internal import agents_pb2


@dataclass(frozen=True)
class InternalAgentServiceConfig:
    host: str = "0.0.0.0"
    port: int = 50061


def _timestamp(value) -> Timestamp:
    result = Timestamp()
    result.FromDatetime(value)
    return result


def _serialize_agent(agent: StoredAgent) -> agents_pb2.AgentInfo:
    return agents_pb2.AgentInfo(
        id=agent.id,
        initial_message=agent.initial_message or "",
        created_at=_timestamp(agent.created_at),
    )


def _serialize_message(message: StoredAgentMessage) -> agents_pb2.AgentMessage:
    return agents_pb2.AgentMessage(
        id=message.id,
        role=message.role,
        content=message.content,
        tool_call_id=message.tool_call_id or "",
        tool_status=message.tool_status or "",
        created_at=_timestamp(message.created_at),
    )


class AgentsServicer:
    def __init__(self, store: SessionStore) -> None:
        self._store = store

    def CreateAgent(self, request, context):  # noqa: N802
        agent = self._store.create_agent(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.CreateAgentResponse(agent=_serialize_agent(agent))

    def ListAgents(self, request, context):  # noqa: N802
        agents = self._store.list_agents(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.ListAgentsResponse(agents=[_serialize_agent(agent) for agent in agents])

    def DescribeAgent(self, request, context):  # noqa: N802
        try:
            agent = self._store.describe_agent(
                org_id=request.org_id,
                user_id=request.user_id,
                canvas_id=request.canvas_id,
                agent_id=request.agent_id,
            )
        except AgentNotFoundError as error:
            context.abort(grpc.StatusCode.NOT_FOUND, "agent not found")
            raise error

        return agents_pb2.DescribeAgentResponse(agent=_serialize_agent(agent))

    def ListAgentMessages(self, request, context):  # noqa: N802
        try:
            messages = self._store.list_messages(
                org_id=request.org_id,
                user_id=request.user_id,
                canvas_id=request.canvas_id,
                agent_id=request.agent_id,
            )
        except AgentNotFoundError as error:
            context.abort(grpc.StatusCode.NOT_FOUND, "agent not found")
            raise error

        return agents_pb2.ListAgentMessagesResponse(messages=[_serialize_message(message) for message in messages])

def add_agents_servicer_to_server(servicer: AgentsServicer, server: grpc.Server) -> None:
    rpc_method_handlers = {
        "CreateAgent": grpc.unary_unary_rpc_method_handler(
            servicer.CreateAgent,
            request_deserializer=agents_pb2.CreateAgentRequest.FromString,
            response_serializer=agents_pb2.CreateAgentResponse.SerializeToString,
        ),
        "ListAgents": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgents,
            request_deserializer=agents_pb2.ListAgentsRequest.FromString,
            response_serializer=agents_pb2.ListAgentsResponse.SerializeToString,
        ),
        "DescribeAgent": grpc.unary_unary_rpc_method_handler(
            servicer.DescribeAgent,
            request_deserializer=agents_pb2.DescribeAgentRequest.FromString,
            response_serializer=agents_pb2.DescribeAgentResponse.SerializeToString,
        ),
        "ListAgentMessages": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgentMessages,
            request_deserializer=agents_pb2.ListAgentMessagesRequest.FromString,
            response_serializer=agents_pb2.ListAgentMessagesResponse.SerializeToString,
        ),
    }

    generic_handler = grpc.method_handlers_generic_handler(
        "Superplane.Internal.Agents.Agents",
        rpc_method_handlers,
    )
    server.add_generic_rpc_handlers((generic_handler,))


class InternalAgentServer:
    def __init__(self, config: InternalAgentServiceConfig, store: SessionStore) -> None:
        self._config = config
        self._store = store
        self._server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
        add_agents_servicer_to_server(AgentsServicer(store), self._server)
        self._server.add_insecure_port(f"{config.host}:{config.port}")
        self._thread: threading.Thread | None = None

    @classmethod
    def from_env(cls, store: SessionStore) -> "InternalAgentServer":
        host = os.getenv("AGENT_INTERNAL_GRPC_HOST", "0.0.0.0").strip() or "0.0.0.0"
        port = int(os.getenv("AGENT_INTERNAL_GRPC_PORT", "50061"))
        return cls(InternalAgentServiceConfig(host=host, port=port), store)

    def start(self) -> None:
        if self._thread is not None and self._thread.is_alive():
            return

        self._server.start()

        def wait_for_termination() -> None:
            self._server.wait_for_termination()

        self._thread = threading.Thread(target=wait_for_termination, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._server.stop(grace=5)
        if self._thread is not None:
            self._thread.join(timeout=5.0)
