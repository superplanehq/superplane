import asyncio
import json
import os
import threading
import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from dataclasses import dataclass
from typing import Any

import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from pydantic_ai import Agent
from pydantic_ai.messages import (
    FinalResultEvent,
    FunctionToolCallEvent,
    FunctionToolResultEvent,
    ModelMessagesTypeAdapter,
    PartDeltaEvent,
    PartStartEvent,
    TextPartDelta,
    ToolCallPart,
    ToolCallPartDelta,
)
from pydantic_ai.models.test import TestModel
from pydantic_ai.run import AgentRunResultEvent

from ai.agent import AgentDeps, build_agent
from ai.grpc import InternalAgentServer
from ai.jwt import JwtClaims, JwtValidator
from ai.persisted_run_recorder import PersistedRunRecorder
from ai.session_store import AgentChatNotFoundError, SessionStore, StoredAgentChat
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from ai.text import normalize_optional


@dataclass(frozen=True)
class WebServerConfig:
    host: str = "127.0.0.1"
    port: int = 8090


class ReplStreamRequest(BaseModel):
    question: str = Field(min_length=1, max_length=2000)
    model: str = Field(
        default=(os.getenv("AI_MODEL", "test").strip() or "test"),
        min_length=1,
        max_length=200,
    )
    base_url: str | None = None


def _debug_enabled() -> bool:
    return os.getenv("REPL_WEB_DEBUG", "").strip().lower() in {"1", "true", "yes", "on"}


def _debug_log(message: str, **fields: Any) -> None:
    if not _debug_enabled():
        return
    if fields:
        field_str = " ".join(f"{key}={_to_jsonable(value)}" for key, value in fields.items())
        print(f"[web] {message} {field_str}", flush=True)
        return
    print(f"[web] {message}", flush=True)


def _resolve_required(value: str | None, env_name: str) -> str:
    resolved = normalize_optional(value) or normalize_optional(os.getenv(env_name))
    if resolved is None:
        raise ValueError(f"Missing required setting: {env_name}")
    return resolved


def _resolve_header(request: Request, header_name: str) -> str | None:
    return normalize_optional(request.headers.get(header_name))


def _resolve_bearer_token(request: Request) -> str | None:
    auth_header = _resolve_header(request, "authorization")
    if auth_header is None:
        return None
    prefix = "bearer "
    if not auth_header.lower().startswith(prefix):
        return None
    return normalize_optional(auth_header[len(prefix) :])


def _resolve_required_bearer_token(request: Request) -> str:
    token = _resolve_bearer_token(request)
    if token is None:
        raise ValueError("Authorization header is required.")
    return token


def _encode_sse_event(data: dict[str, Any]) -> str:
    serialized = json.dumps(data, separators=(",", ":"))
    return f"data: {serialized}\n\n"


def _iter_text_chunks(text: str, chunk_size: int = 28) -> list[str]:
    if not text:
        return []
    return [text[index : index + chunk_size] for index in range(0, len(text), chunk_size)]


def _to_jsonable(value: Any) -> Any:
    if value is None:
        return None
    if isinstance(value, (str, int, float, bool)):
        return value
    if isinstance(value, dict):
        return {str(key): _to_jsonable(item) for key, item in value.items()}
    if isinstance(value, list):
        return [_to_jsonable(item) for item in value]
    model_dump = getattr(value, "model_dump", None)
    if callable(model_dump):
        return model_dump(mode="json", by_alias=True)
    return str(value)


def _load_message_history(store: SessionStore, chat_id: str) -> Any:
    history = store.load_agent_chat_message_history(chat_id)
    if not history:
        return None
    return ModelMessagesTypeAdapter.validate_python(history)


def _resolve_agent_context(chat_id: str, request: Request) -> tuple[JwtClaims, StoredAgentChat]:
    api_token = _resolve_required_bearer_token(request)
    jwt_validator = JwtValidator.from_env()
    claims = jwt_validator.decode(api_token)
    store: SessionStore = request.app.state.session_store
    chat = store.get_agent_chat(chat_id)
    if chat.org_id != claims.org_id or chat.user_id != claims.subject:
        raise ValueError("Scoped token does not allow the requested agent.")
    jwt_validator.validate_canvas(chat.canvas_id, claims)
    return claims, chat


def _build_deps(payload: ReplStreamRequest, request: Request, claims: JwtClaims, canvas_id: str) -> AgentDeps:
    base_url = _resolve_required(payload.base_url, "SUPERPLANE_BASE_URL")
    api_token = _resolve_required_bearer_token(request)
    client = SuperplaneClient(
        SuperplaneClientConfig(
            base_url=base_url,
            api_token=api_token,
            organization_id=claims.org_id,
        )
    )
    _debug_log(
        "resolved non-test deps",
        model=payload.model,
        canvas_id=canvas_id,
        base_url=base_url,
        organization_id=claims.org_id,
        has_token=bool(api_token),
    )
    return AgentDeps(
        client=client,
        canvas_id=canvas_id,
    )


async def _run_stream_events(agent: Any, **kwargs: Any) -> AsyncIterator[Any]:
    async for event in agent.run_stream_events(**kwargs):
        yield event


async def _stream_agent_run(chat_id: str, payload: ReplStreamRequest, request: Request) -> AsyncIterator[dict[str, Any]]:
    started_at = time.perf_counter()
    store: SessionStore = request.app.state.session_store
    claims: JwtClaims | None = None
    if payload.model == "test" and _resolve_bearer_token(request) is None:
        try:
            chat = store.get_agent_chat(chat_id)
        except AgentChatNotFoundError:
            chat = store.create_agent_chat(
                org_id="test-org",
                user_id="test-user",
                canvas_id="test-canvas",
                chat_id=chat_id,
            )
    else:
        claims, chat = _resolve_agent_context(chat_id, request)

    message_history = _load_message_history(store, chat.id)
    recorder = PersistedRunRecorder(store, chat.id, payload.question)
    resolved_canvas_id = chat.canvas_id
    deps: AgentDeps | None = None
    if payload.model != "test":
        if claims is None:
            raise ValueError("Agent claims are missing.")
        deps = _build_deps(payload, request, claims, resolved_canvas_id)

    _debug_log(
        "starting agent run",
        chat_id=chat.id,
        model=payload.model,
        canvas_id=resolved_canvas_id,
        question_preview=payload.question[:120],
        has_history=message_history is not None,
    )
    yield {
        "type": "run_started",
        "chat_id": chat.id,
        "model": payload.model,
        "canvas_id": resolved_canvas_id,
    }

    run_kwargs: dict[str, Any] = {"user_prompt": payload.question}
    if message_history is not None:
        run_kwargs["message_history"] = message_history

    if payload.model == "test":
        _debug_log("using test model run path", canvas_id=resolved_canvas_id, chat_id=chat.id)
        test_agent: Agent[None, str] = Agent(model=TestModel(), output_type=str)
        async for event in _run_stream_events(test_agent, **run_kwargs):
            if isinstance(event, PartDeltaEvent) and isinstance(event.delta, TextPartDelta):
                chunk = event.delta.content_delta
                if chunk:
                    recorder.append_assistant_content(chunk)
                    yield {
                        "type": "model_delta",
                        "content": chunk,
                    }
                continue

            if isinstance(event, AgentRunResultEvent):
                result = event.result
                recorder.save_authoritative_messages(result.new_messages())
                output = _to_jsonable(result.output)
                if isinstance(output, str) and output:
                    recorder.set_assistant_content(output)
                yield {
                    "type": "final_answer",
                    "output": output,
                    "usage": _to_jsonable(result.usage()),
                }

        yield {
            "type": "run_completed",
            "elapsed_ms": (time.perf_counter() - started_at) * 1000,
        }
        yield {"type": "done"}
        return

    agent = build_agent(model=payload.model)
    if deps is None:
        raise ValueError("Agent dependencies are missing.")

    run_kwargs["deps"] = deps
    _debug_log(
        "running non-test agent",
        chat_id=chat.id,
        model=payload.model,
        canvas_id=resolved_canvas_id,
    )

    output_tool_call_id: str | None = None
    output_tool_name_hints = {"final_result", "return_canvasanswer", "canvasanswer"}
    output_args_buffer_by_call_id: dict[str, str] = {}
    streamed_answer_length_by_call_id: dict[str, int] = {}
    streamed_any_answer_delta = False

    def emit_answer_delta_from_output_args(call_id: str, output_args: Any) -> dict[str, Any] | None:
        nonlocal streamed_any_answer_delta
        if not isinstance(output_args, dict):
            return None
        answer = output_args.get("answer")
        if not isinstance(answer, str):
            return None
        already_streamed = streamed_answer_length_by_call_id.get(call_id, 0)
        if len(answer) <= already_streamed:
            return None
        delta = answer[already_streamed:]
        streamed_answer_length_by_call_id[call_id] = len(answer)
        streamed_any_answer_delta = True
        recorder.append_assistant_content(delta)
        return {
            "type": "model_delta",
            "content": delta,
        }

    def likely_output_tool_name(tool_name: str | None) -> bool:
        if not isinstance(tool_name, str):
            return False
        normalized = tool_name.strip().lower()
        return normalized in output_tool_name_hints

    tool_started_at_by_call_id: dict[str, float] = {}
    async for event in _run_stream_events(agent, **run_kwargs):
        if isinstance(event, PartStartEvent) and isinstance(event.part, ToolCallPart):
            tool_call_id = event.part.tool_call_id
            if tool_call_id and likely_output_tool_name(event.part.tool_name):
                output_tool_call_id = tool_call_id
            recorder.tool_started(event.part)
            continue

        if isinstance(event, FinalResultEvent):
            if event.tool_call_id:
                output_tool_call_id = event.tool_call_id
            continue

        if isinstance(event, PartDeltaEvent) and isinstance(event.delta, ToolCallPartDelta):
            tool_call_id = event.delta.tool_call_id
            if tool_call_id is None:
                continue

            if output_tool_call_id is None and likely_output_tool_name(event.delta.tool_name_delta):
                output_tool_call_id = tool_call_id
            if output_tool_call_id != tool_call_id:
                continue

            args_delta = event.delta.args_delta
            if isinstance(args_delta, dict):
                recorder.tool_call_delta(tool_call_id, args_delta, event.delta.tool_name_delta)
                maybe_delta = emit_answer_delta_from_output_args(tool_call_id, args_delta)
                if maybe_delta is not None:
                    yield maybe_delta
                continue

            if isinstance(args_delta, str):
                recorder.tool_call_delta(tool_call_id, args_delta, event.delta.tool_name_delta)
                buffer = output_args_buffer_by_call_id.get(tool_call_id, "")
                buffer += args_delta
                output_args_buffer_by_call_id[tool_call_id] = buffer
                try:
                    parsed_args = json.loads(buffer)
                except json.JSONDecodeError:
                    continue
                maybe_delta = emit_answer_delta_from_output_args(tool_call_id, parsed_args)
                if maybe_delta is not None:
                    yield maybe_delta
            continue

        if isinstance(event, FunctionToolCallEvent):
            tool_call_id = event.part.tool_call_id or event.part.tool_name
            tool_started_at_by_call_id[tool_call_id] = time.perf_counter()
            yield {
                "type": "tool_started",
                "tool_name": event.part.tool_name,
                "tool_call_id": tool_call_id,
                "args": _to_jsonable(event.part.args),
            }
            continue

        if isinstance(event, FunctionToolResultEvent):
            tool_call_id = event.result.tool_call_id or event.result.tool_name
            tool_started_at = tool_started_at_by_call_id.pop(tool_call_id, started_at)
            elapsed_ms = (time.perf_counter() - tool_started_at) * 1000
            recorder.tool_finished(event)
            yield {
                "type": "tool_finished",
                "tool_name": event.result.tool_name,
                "tool_call_id": tool_call_id,
                "elapsed_ms": elapsed_ms,
            }
            continue

        if isinstance(event, PartDeltaEvent) and isinstance(event.delta, TextPartDelta):
            chunk = event.delta.content_delta
            if chunk:
                recorder.append_assistant_content(chunk)
                yield {
                    "type": "model_delta",
                    "content": chunk,
                }
            continue

        if isinstance(event, AgentRunResultEvent):
            result = event.result
            recorder.save_authoritative_messages(result.new_messages())
            output = _to_jsonable(result.output)
            if isinstance(output, dict) and not streamed_any_answer_delta:
                answer = output.get("answer")
                if isinstance(answer, str) and answer:
                    recorder.set_assistant_content(answer)
                    for chunk in _iter_text_chunks(answer):
                        yield {
                            "type": "model_delta",
                            "content": chunk,
                        }
                        await asyncio.sleep(0.01)
            elif isinstance(output, str) and output:
                recorder.set_assistant_content(output)
            yield {
                "type": "final_answer",
                "output": output,
                "usage": _to_jsonable(result.usage()),
            }

    yield {
        "type": "run_completed",
        "elapsed_ms": (time.perf_counter() - started_at) * 1000,
    }
    yield {"type": "done"}


def _create_app() -> FastAPI:
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        store = SessionStore()
        app.state.session_store = store
        grpc_server = InternalAgentServer.from_env(store)
        grpc_server.start()
        app.state.internal_agent_server = grpc_server
        try:
            yield
        finally:
            grpc_server.stop()
            store.close()

    app = FastAPI(lifespan=lifespan)
    cors_origins_raw = os.getenv("REPL_WEB_CORS_ORIGINS", "*")
    cors_origins = [origin.strip() for origin in cors_origins_raw.split(",") if origin.strip()]
    if not cors_origins:
        cors_origins = ["*"]

    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.post("/agents/chats/{chat_id}/stream")
    async def stream_repl(chat_id: str, payload: ReplStreamRequest, request: Request) -> StreamingResponse:
        if payload.model != "test" and _resolve_bearer_token(request) is None:
            raise HTTPException(status_code=401, detail="Authorization header is required")

        _debug_log(
            "incoming stream request",
            chat_id=chat_id,
            model=payload.model,
            has_base_url=bool(normalize_optional(payload.base_url) or normalize_optional(os.getenv("SUPERPLANE_BASE_URL"))),
            has_token=bool(_resolve_bearer_token(request)),
        )

        async def event_generator() -> AsyncIterator[str]:
            try:
                async for event in _stream_agent_run(chat_id, payload, request):
                    if await request.is_disconnected():
                        _debug_log("client disconnected", chat_id=chat_id)
                        break
                    yield _encode_sse_event(event)
            except Exception as error:
                _debug_log("stream failed", chat_id=chat_id, error=str(error))
                yield _encode_sse_event(
                    {
                        "type": "run_failed",
                        "error": str(error),
                    }
                )
                yield _encode_sse_event({"type": "done"})

        return StreamingResponse(
            event_generator(),
            media_type="text/event-stream",
            headers={
                "cache-control": "no-cache",
                "connection": "keep-alive",
            },
        )

    return app


def create_app() -> FastAPI:
    return _create_app()


class WebServer:
    def __init__(self, config: WebServerConfig) -> None:
        self._config = config
        self._app = _create_app()
        self._uvicorn_config = uvicorn.Config(
            self._app,
            host=config.host,
            port=config.port,
            log_level="warning",
            access_log=False,
            ws="none",
        )
        self._server = uvicorn.Server(self._uvicorn_config)
        self._thread: threading.Thread | None = None
        self._startup_error: BaseException | None = None

    @property
    def base_url(self) -> str:
        host = self._config.host
        if host == "0.0.0.0":
            host = "127.0.0.1"
        return f"http://{host}:{self._config.port}"

    def start(self) -> None:
        if self._thread is not None and self._thread.is_alive():
            return
        self._startup_error = None
        self._thread = threading.Thread(target=self.serve_forever, daemon=True)
        self._thread.start()
        for _ in range(200):
            if self._server.started:
                return
            if not self._thread.is_alive():
                break
            time.sleep(0.01)
        error = self._startup_error
        raise RuntimeError(
            f"Failed to start REPL web server at {self.base_url}. "
            "Check whether the port is already in use."
        ) from error

    def stop(self) -> None:
        self._server.should_exit = True
        if self._thread is not None:
            self._thread.join(timeout=5.0)

    def serve_forever(self) -> None:
        try:
            self._server.run()
        except SystemExit as error:
            self._startup_error = error
