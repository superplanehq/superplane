import os
import threading
import uuid

from concurrent import futures
from dataclasses import dataclass

import grpc
from google.protobuf.timestamp_pb2 import Timestamp

from ai.session_store import AgentChatNotFoundError, SessionStore, StoredAgentChat, StoredAgentChatMessage
from private import agents_pb2


@dataclass(frozen=True)
class AgentServiceConfig:
    host: str = "0.0.0.0"
    port: int = 50061


def _timestamp(value) -> Timestamp:
    result = Timestamp()
    result.FromDatetime(value)
    return result


def _serialize_chat(chat: StoredAgentChat) -> agents_pb2.ChatInfo:
    return agents_pb2.ChatInfo(
        id=chat.id,
        initial_message=chat.initial_message or "",
        created_at=_timestamp(chat.created_at),
    )


def _serialize_message(message: StoredAgentChatMessage) -> agents_pb2.AgentChatMessage:
    return agents_pb2.AgentChatMessage(
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

    def CreateAgentChat(self, request, context):  # noqa: N802
        chat = self._store.create_agent_chat(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.CreateAgentChatResponse(chat=_serialize_chat(chat))

    def ListAgentChats(self, request, context):  # noqa: N802
        chats = self._store.list_agent_chats(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.ListAgentChatsResponse(chats=[_serialize_chat(chat) for chat in chats])

    def DescribeAgentChat(self, request, context):  # noqa: N802
        try:
            chat = self._store.describe_agent_chat(
                org_id=request.org_id,
                user_id=request.user_id,
                canvas_id=request.canvas_id,
                chat_id=request.chat_id,
            )
        except AgentChatNotFoundError as error:
            context.abort(grpc.StatusCode.NOT_FOUND, "chat not found")
            raise error

        return agents_pb2.DescribeAgentChatResponse(chat=_serialize_chat(chat))

    def ListAgentChatMessages(self, request, context):  # noqa: N802
        try:
            messages = self._store.list_agent_chat_messages(
                org_id=request.org_id,
                user_id=request.user_id,
                canvas_id=request.canvas_id,
                chat_id=request.chat_id,
            )
        except AgentChatNotFoundError as error:
            context.abort(grpc.StatusCode.NOT_FOUND, "chat not found")
            raise error

        return agents_pb2.ListAgentChatMessagesResponse(messages=[_serialize_message(message) for message in messages])

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
    def __init__(self, config: AgentServiceConfig, store: SessionStore) -> None:
        self._config = config
        self._store = store
        self._server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
        add_agents_servicer_to_server(AgentsServicer(store), self._server)
        self._server.add_insecure_port(f"{config.host}:{config.port}")
        self._thread: threading.Thread | None = None

    @classmethod
    def from_env(cls, store: SessionStore) -> "InternalAgentServer":
        host = os.getenv("INTERNAL_GRPC_HOST", "0.0.0.0").strip() or "0.0.0.0"
        port = int(os.getenv("INTERNAL_GRPC_PORT", "50061"))
        return cls(AgentServiceConfig(host=host, port=port), store)

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
