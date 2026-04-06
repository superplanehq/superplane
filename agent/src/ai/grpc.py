import threading
from concurrent import futures
from dataclasses import dataclass
from datetime import datetime
from typing import Any

import grpc  # type: ignore[import-untyped]
from google.protobuf.timestamp_pb2 import Timestamp

from ai.config import config
from ai.session_store import (
    AgentChatNotFoundError,
    SessionStore,
    StoredAgentChat,
    StoredAgentChatMessage,
)
from private import agents_pb2


@dataclass(frozen=True)
class AgentServiceConfig:
    host: str = "0.0.0.0"
    port: int = 50061


def _timestamp(value: datetime) -> Timestamp:
    result = Timestamp()
    result.FromDatetime(value)
    return result


def _serialize_chat_usage(chat: StoredAgentChat) -> Any:
    return agents_pb2.ChatUsage(  # type: ignore[attr-defined]
        total_input_tokens=chat.total_input_tokens,
        total_output_tokens=chat.total_output_tokens,
        total_tokens=chat.total_tokens,
        total_estimated_cost_usd=chat.total_estimated_cost_usd,
    )


def _serialize_chat(chat: StoredAgentChat) -> Any:
    return agents_pb2.ChatInfo(  # type: ignore[attr-defined]
        id=chat.id,
        initial_message=chat.initial_message or "",
        created_at=_timestamp(chat.created_at),
        usage=_serialize_chat_usage(chat),
    )


def _serialize_message(message: StoredAgentChatMessage) -> Any:
    return agents_pb2.AgentChatMessage(  # type: ignore[attr-defined]
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

    def CreateAgentChat(self, request: Any, context: Any) -> Any:  # noqa: N802
        chat = self._store.create_agent_chat(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.CreateAgentChatResponse(chat=_serialize_chat(chat))  # type: ignore[attr-defined]

    def ListAgentChats(self, request: Any, context: Any) -> Any:  # noqa: N802
        chats = self._store.list_agent_chats(
            org_id=request.org_id,
            user_id=request.user_id,
            canvas_id=request.canvas_id,
        )
        return agents_pb2.ListAgentChatsResponse(chats=[_serialize_chat(chat) for chat in chats])  # type: ignore[attr-defined]

    def DescribeAgentChat(self, request: Any, context: Any) -> Any:  # noqa: N802
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

        return agents_pb2.DescribeAgentChatResponse(chat=_serialize_chat(chat))  # type: ignore[attr-defined]

    def ListAgentChatMessages(self, request: Any, context: Any) -> Any:  # noqa: N802
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

        return agents_pb2.ListAgentChatMessagesResponse(  # type: ignore[attr-defined]
            messages=[_serialize_message(message) for message in messages]
        )

    def DescribeOrganizationAgentUsage(self, request: Any, context: Any) -> Any:  # noqa: N802
        try:
            usage = self._store.get_org_usage(org_id=request.org_id)
        except Exception as error:
            print(f"[agent] failed to load org usage for org {request.org_id}: {error}", flush=True)
            context.abort(grpc.StatusCode.UNAVAILABLE, "failed to load organization usage")
            return agents_pb2.DescribeOrganizationAgentUsageResponse()  # type: ignore[attr-defined]

        return agents_pb2.DescribeOrganizationAgentUsageResponse(  # type: ignore[attr-defined]
            usage=agents_pb2.ChatUsage(  # type: ignore[attr-defined]
                total_input_tokens=usage.total_input_tokens,
                total_output_tokens=usage.total_output_tokens,
                total_tokens=usage.total_tokens,
                total_estimated_cost_usd=usage.total_estimated_cost_usd,
            )
        )


def add_agents_servicer_to_server(servicer: AgentsServicer, server: grpc.Server) -> None:
    rpc_method_handlers = {
        "CreateAgentChat": grpc.unary_unary_rpc_method_handler(
            servicer.CreateAgentChat,
            request_deserializer=agents_pb2.CreateAgentChatRequest.FromString,  # type: ignore[attr-defined]
            response_serializer=agents_pb2.CreateAgentChatResponse.SerializeToString,  # type: ignore[attr-defined]
        ),
        "ListAgentChats": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgentChats,
            request_deserializer=agents_pb2.ListAgentChatsRequest.FromString,  # type: ignore[attr-defined]
            response_serializer=agents_pb2.ListAgentChatsResponse.SerializeToString,  # type: ignore[attr-defined]
        ),
        "DescribeAgentChat": grpc.unary_unary_rpc_method_handler(
            servicer.DescribeAgentChat,
            request_deserializer=agents_pb2.DescribeAgentChatRequest.FromString,  # type: ignore[attr-defined]
            response_serializer=agents_pb2.DescribeAgentChatResponse.SerializeToString,  # type: ignore[attr-defined]
        ),
        "ListAgentChatMessages": grpc.unary_unary_rpc_method_handler(
            servicer.ListAgentChatMessages,
            request_deserializer=agents_pb2.ListAgentChatMessagesRequest.FromString,  # type: ignore[attr-defined]
            response_serializer=agents_pb2.ListAgentChatMessagesResponse.SerializeToString,  # type: ignore[attr-defined]
        ),
        "DescribeOrganizationAgentUsage": grpc.unary_unary_rpc_method_handler(
            servicer.DescribeOrganizationAgentUsage,
            request_deserializer=agents_pb2.DescribeOrganizationAgentUsageRequest.FromString,  # type: ignore[attr-defined]
            response_serializer=agents_pb2.DescribeOrganizationAgentUsageResponse.SerializeToString,  # type: ignore[attr-defined]
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
        return cls(AgentServiceConfig(host=config.grpc_host, port=config.grpc_port), store)

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
