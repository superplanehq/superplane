import asyncio
import json
import random
import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import Response, StreamingResponse
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
from pydantic_ai.usage import RunUsage

from ai.agent import AgentDeps, build_agent
from ai.config import config
from ai.grpc import InternalAgentServer
from ai.jwt import JwtClaims, JwtValidator
from ai.memory import (
    curate_canvas_memory_markdown,
    register_background_task,
    snippet_from_run_output,
)
from ai.models import CanvasAnswer
from ai.persisted_run_recorder import PersistedRunRecorder
from ai.proposal_configuration_coerce import coerce_canvas_answer_proposal
from ai.session_store import (
    AgentChatNotFoundError,
    SessionStore,
    StoredAgentChat,
    apply_tool_display_labels_to_messages,
)
from ai.stream_tracker import ActiveStreamTracker
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from ai.telemetry import init_sentry, init_telemetry, shutdown_sentry, shutdown_telemetry
from ai.text import normalize_optional
from ai.tools import format_tool_display_label
from ai.usage_limit_checker import (
    AgentUsageLimitChecker,
    NoopUsageLimitChecker,
    UsageLimitChecker,
)
from ai.usage_publisher import AgentUsagePublisher, NoopUsagePublisher, UsagePublisher


class AgentStreamRequest(BaseModel):
    question: str = Field(min_length=1, max_length=2000)
    model: str = Field(
        default=config.ai_model,
        min_length=1,
        max_length=200,
    )
    base_url: str | None = None


def _debug_enabled() -> bool:
    return config.debug


def _debug_log(message: str, **fields: Any) -> None:
    if not _debug_enabled():
        return
    if fields:
        field_str = " ".join(f"{key}={_to_jsonable(value)}" for key, value in fields.items())
        print(f"[web] {message} {field_str}", flush=True)
        return
    print(f"[web] {message}", flush=True)


def _resolve_required(value: str | None, fallback: str | None, env_name: str) -> str:
    resolved = normalize_optional(value) or normalize_optional(fallback)
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


def _stable_tool_call_id(raw_call_id: str | None, tool_name: str | None) -> str:
    if isinstance(raw_call_id, str):
        stripped = raw_call_id.strip()
        if stripped:
            return stripped
    name = (tool_name or "").strip()
    return name if name else "tool"


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


def _record_usage(
    store: SessionStore,
    publisher: AgentUsagePublisher,
    run_id: str,
    usage: RunUsage,
    org_id: str,
    chat_id: str,
    model: str,
) -> None:
    try:
        store.update_run_usage(
            run_id=run_id,
            input_tokens=usage.input_tokens or 0,
            output_tokens=usage.output_tokens or 0,
            cache_read_tokens=usage.cache_read_tokens or 0,
            cache_write_tokens=usage.cache_write_tokens or 0,
            total_tokens=usage.total_tokens or 0,
        )
    except Exception as error:
        print(f"[web] failed to record usage for run {run_id}: {error}", flush=True)

    # DB write and RabbitMQ publish are independent
    # a DB failure won't prevent publishing, and each has its own error logging
    try:
        publisher.publish_agent_run_finished(
            organization_id=org_id,
            chat_id=chat_id,
            model=model,
            input_tokens=usage.input_tokens or 0,
            output_tokens=usage.output_tokens or 0,
            total_tokens=usage.total_tokens or 0,
        )
    except Exception as error:
        print(f"[web] failed to publish usage for run {run_id}: {error}", flush=True)


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


def _build_deps(
    payload: AgentStreamRequest,
    request: Request,
    claims: JwtClaims,
    canvas_id: str,
    session_store: SessionStore,
) -> AgentDeps:
    base_url = _resolve_required(
        payload.base_url, config.superplane_base_url, "SUPERPLANE_BASE_URL"
    )
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
        session_store=session_store,
    )


_TRANSIENT_STATUS_CODES = {429, 502, 503, 504, 529}
_MAX_ATTEMPTS = 3
_BASE_DELAY_SECONDS = 1.0
_JITTER_SECONDS = 0.5


def _extract_status_code(error: BaseException) -> int | None:
    status_code = getattr(error, "status_code", None)
    if isinstance(status_code, int):
        return status_code
    cause = error.__cause__ or error.__context__
    if cause is not None:
        cause_status = getattr(cause, "status_code", None)
        if isinstance(cause_status, int):
            return cause_status
    return None


def _is_transient_error(error: Exception) -> bool:
    status_code = _extract_status_code(error)
    return status_code is not None and status_code in _TRANSIENT_STATUS_CODES


def _friendly_error_message(error: Exception) -> str:
    status_code = _extract_status_code(error)
    if status_code == 529:
        return "The AI service is temporarily overloaded. Please try again in a moment."
    if status_code == 429:
        return "Rate limit reached. Please wait a moment and try again."
    if status_code in {502, 503, 504}:
        return "The AI service is temporarily unavailable. Please try again in a moment."
    if isinstance(status_code, int) and 400 <= status_code < 500:
        return "AI service configuration error. Please contact support."
    return "An unexpected error occurred. Please try again."


async def _run_stream_events(agent: Any, **kwargs: Any) -> AsyncIterator[Any]:
    yielded_any = False
    for attempt in range(_MAX_ATTEMPTS):
        try:
            async for event in agent.run_stream_events(**kwargs):
                yielded_any = True
                yield event
            return
        except Exception as error:
            if yielded_any or not _is_transient_error(error) or attempt == _MAX_ATTEMPTS - 1:
                raise
            delay = _BASE_DELAY_SECONDS * (2**attempt) + random.uniform(0, _JITTER_SECONDS)
            _debug_log(
                "transient LLM error, retrying",
                attempt=attempt + 1,
                delay=delay,
                error=str(error),
            )
            await asyncio.sleep(delay)


async def _stream_agent_run(
    chat_id: str, payload: AgentStreamRequest, request: Request
) -> AsyncIterator[dict[str, Any]]:
    started_at = time.perf_counter()
    store: SessionStore = request.app.state.session_store
    publisher: AgentUsagePublisher = request.app.state.publisher
    limit_checker: AgentUsageLimitChecker = request.app.state.limit_checker
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

    await limit_checker.check_agent_token_limit(chat.org_id)

    message_history = _load_message_history(store, chat.id)
    resolved_canvas_id = chat.canvas_id
    deps: AgentDeps | None = None
    if payload.model != "test":
        if claims is None:
            raise ValueError("Agent claims are missing.")
        deps = _build_deps(payload, request, claims, resolved_canvas_id, store)

    run_id = store.create_agent_chat_run(chat.id, payload.model)
    recorder = PersistedRunRecorder(store, chat.id, run_id, payload.question)

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

    persisted_tool_display_labels: dict[str, str] = {}

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
                messages = apply_tool_display_labels_to_messages(
                    list(result.new_messages()),
                    persisted_tool_display_labels,
                )
                recorder.save_authoritative_messages(messages)
                output = _to_jsonable(result.output)
                if isinstance(output, str) and output:
                    recorder.set_assistant_content(output)
                usage = result.usage()
                _record_usage(store, publisher, run_id, usage, chat.org_id, chat.id, payload.model)
                yield {
                    "type": "final_answer",
                    "output": output,
                    "usage": _to_jsonable(usage),
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

    last_assistant_snippet = ""
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
    tool_display_label_by_call_id: dict[str, str] = {}
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
            _raw_tool_call_id = event.delta.tool_call_id
            if _raw_tool_call_id is None:
                continue
            tool_call_id = _raw_tool_call_id

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
            tool_call_id = _stable_tool_call_id(event.part.tool_call_id, event.part.tool_name)
            tool_started_at_by_call_id[tool_call_id] = time.perf_counter()
            tool_label = format_tool_display_label(
                event.part.tool_name or "",
                event.part.args,
                deps,
            )
            tool_display_label_by_call_id[tool_call_id] = tool_label
            yield {
                "type": "tool_started",
                "tool_name": event.part.tool_name,
                "tool_call_id": tool_call_id,
                "tool_label": tool_label,
                "args": _to_jsonable(event.part.args),
            }
            continue

        if isinstance(event, FunctionToolResultEvent):
            tool_call_id = _stable_tool_call_id(event.result.tool_call_id, event.result.tool_name)
            tool_started_at = tool_started_at_by_call_id.pop(tool_call_id, started_at)
            elapsed_ms = (time.perf_counter() - tool_started_at) * 1000
            tool_label = tool_display_label_by_call_id.pop(
                tool_call_id,
                (event.result.tool_name or "").strip() or "tool",
            )
            persisted_tool_display_labels[tool_call_id] = tool_label
            recorder.tool_finished(event)
            yield {
                "type": "tool_finished",
                "tool_name": event.result.tool_name,
                "tool_call_id": tool_call_id,
                "tool_label": tool_label,
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
            messages = apply_tool_display_labels_to_messages(
                list(result.new_messages()),
                persisted_tool_display_labels,
            )
            recorder.save_authoritative_messages(messages)
            resolved_output = result.output
            if isinstance(resolved_output, CanvasAnswer):
                canvas_summary = deps.canvas_cache.get(deps.canvas_id)
                resolved_output = coerce_canvas_answer_proposal(
                    deps.client,
                    resolved_output,
                    canvas_summary,
                )
            output = _to_jsonable(resolved_output)
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
            usage = result.usage()
            _record_usage(store, publisher, run_id, usage, chat.org_id, chat.id, payload.model)
            yield {
                "type": "final_answer",
                "output": output,
                "usage": _to_jsonable(usage),
            }
            last_assistant_snippet = snippet_from_run_output(resolved_output)

    if last_assistant_snippet.strip():
        register_background_task(
            asyncio.create_task(
                curate_canvas_memory_markdown(
                    store=store,
                    canvas_id=resolved_canvas_id,
                    model=payload.model,
                    user_question=payload.question,
                    assistant_reply=last_assistant_snippet,
                )
            )
        )

    yield {
        "type": "run_completed",
        "elapsed_ms": (time.perf_counter() - started_at) * 1000,
    }
    yield {"type": "done"}


def _create_app() -> FastAPI:
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        init_sentry()
        init_telemetry()
        store = SessionStore()
        tracker = ActiveStreamTracker()
        rabbitmq_url = config.rabbitmq_url
        publisher: AgentUsagePublisher = (
            UsagePublisher(rabbitmq_url) if rabbitmq_url else NoopUsagePublisher()
        )
        usage_grpc_url = config.usage_grpc_url
        limit_checker: AgentUsageLimitChecker = (
            UsageLimitChecker(usage_grpc_url) if usage_grpc_url else NoopUsageLimitChecker()
        )
        app.state.session_store = store
        app.state.stream_tracker = tracker
        app.state.publisher = publisher
        app.state.limit_checker = limit_checker
        grpc_server = InternalAgentServer.from_env(store)
        grpc_server.start()
        app.state.internal_agent_server = grpc_server
        try:
            yield
        finally:
            tracker.begin_shutdown()
            await tracker.wait_for_drain()
            grpc_server.stop()
            publisher.close()
            await limit_checker.close()
            store.close()
            shutdown_telemetry()
            shutdown_sentry()

    app = FastAPI(lifespan=lifespan)
    cors_origins = [origin.strip() for origin in config.cors_origins.split(",") if origin.strip()]
    if not cors_origins:
        cors_origins = ["*"]

    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/health")
    async def health() -> Response:
        # Match pkg/public Server.HealthCheck: 200 with empty body for load balancers / probes.
        return Response(status_code=200)

    @app.post("/agents/chats/{chat_id}/stream")
    async def stream_agent_chat(
        chat_id: str, payload: AgentStreamRequest, request: Request
    ) -> StreamingResponse:
        tracker: ActiveStreamTracker = request.app.state.stream_tracker
        if tracker.is_shutting_down:
            raise HTTPException(status_code=503, detail="Service is shutting down")

        if payload.model != "test" and _resolve_bearer_token(request) is None:
            raise HTTPException(status_code=401, detail="Authorization header is required")

        _debug_log(
            "incoming stream request",
            chat_id=chat_id,
            model=payload.model,
            has_base_url=bool(
                normalize_optional(payload.base_url)
                or normalize_optional(config.superplane_base_url)
            ),
            has_token=bool(_resolve_bearer_token(request)),
        )

        await tracker.acquire()
        try:

            async def event_generator() -> AsyncIterator[str]:
                try:
                    async for event in _stream_agent_run(chat_id, payload, request):
                        if await request.is_disconnected():
                            _debug_log("client disconnected", chat_id=chat_id)
                            break
                        yield _encode_sse_event(event)
                except Exception as error:
                    import sentry_sdk

                    sentry_sdk.capture_exception(error)
                    print(f"[web] stream failed chat_id={chat_id} error={error}", flush=True)
                    yield _encode_sse_event(
                        {
                            "type": "run_failed",
                            "error": _friendly_error_message(error),
                        }
                    )
                    yield _encode_sse_event({"type": "done"})
                finally:
                    await tracker.release()

            return StreamingResponse(
                event_generator(),
                media_type="text/event-stream",
                headers={
                    "cache-control": "no-cache",
                    "connection": "keep-alive",
                },
            )
        except BaseException:
            await tracker.release()
            raise

    return app


def create_app() -> FastAPI:
    return _create_app()
