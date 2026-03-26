import os
import threading
import uuid

from concurrent import futures
from dataclasses import dataclass

import grpc

from private import agents_pb2


@dataclass(frozen=True)
class AgentServiceConfig:
    host: str = "0.0.0.0"
    port: int = 50061


class AgentsServicer:
    def __init__(self) -> None:
        pass

    def CreateAgentChat(self, request, context):  # noqa: N802
        #
        # TODO: just return a random UUID for now.
        # We should update this to create an agent chat record, and returns its ID.
        #
        return agents_pb2.CreateAgentChatResponse(
            chat=agents_pb2.ChatInfo(id=uuid.uuid4().hex)
        )

    def ListAgentChats(self, request, context):  # noqa: N802
        context.abort(grpc.StatusCode.UNIMPLEMENTED, "not implemented")

    def DescribeAgentChat(self, request, context):  # noqa: N802
        context.abort(grpc.StatusCode.UNIMPLEMENTED, "not implemented")

    def ListAgentChatMessages(self, request, context):  # noqa: N802
        context.abort(grpc.StatusCode.UNIMPLEMENTED, "not implemented")


def add_agents_servicer_to_server(servicer: AgentsServicer, server: grpc.Server) -> None:
    rpc_method_handlers = {
        "CreateAgentChat": grpc.unary_unary_rpc_method_handler(
            servicer.CreateAgentChat,
            request_deserializer=agents_pb2.CreateAgentChatRequest.FromString,
            response_serializer=agents_pb2.CreateAgentChatResponse.SerializeToString,
        ),
        "ListAgentChats": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgentChats,
            request_deserializer=agents_pb2.ListAgentChatsRequest.FromString,
            response_serializer=agents_pb2.ListAgentChatsResponse.SerializeToString,
        ),
        "DescribeAgentChat": grpc.unary_unary_rpc_method_handler(
            servicer.DescribeAgentChat,
            request_deserializer=agents_pb2.DescribeAgentChatRequest.FromString,
            response_serializer=agents_pb2.DescribeAgentChatResponse.SerializeToString,
        ),
        "ListAgentChatMessages": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgentChatMessages,
            request_deserializer=agents_pb2.ListAgentChatMessagesRequest.FromString,
            response_serializer=agents_pb2.ListAgentChatMessagesResponse.SerializeToString,
        ),
    }

    generic_handler = grpc.method_handlers_generic_handler(
        "Superplane.Internal.Agents.Agents",
        rpc_method_handlers,
    )
    server.add_generic_rpc_handlers((generic_handler,))


class InternalAgentServer:
    def __init__(self, config: AgentServiceConfig) -> None:
        self._config = config
        self._server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
        add_agents_servicer_to_server(AgentsServicer(), self._server)
        self._server.add_insecure_port(f"{config.host}:{config.port}")
        self._thread: threading.Thread | None = None

    @classmethod
    def from_env(cls) -> "InternalAgentServer":
        host = os.getenv("INTERNAL_GRPC_HOST", "0.0.0.0").strip() or "0.0.0.0"
        port = int(os.getenv("INTERNAL_GRPC_PORT", "50061"))
        return cls(AgentServiceConfig(host=host, port=port))

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
