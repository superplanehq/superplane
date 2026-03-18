import asyncio
import json
import os
import threading
import time
from collections.abc import AsyncIterator
from dataclasses import dataclass
from typing import Any

import uvicorn
from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from pydantic_ai import Agent
from pydantic_ai.messages import (
    FinalResultEvent,
    FunctionToolCallEvent,
    FunctionToolResultEvent,
    PartDeltaEvent,
    PartStartEvent,
    TextPartDelta,
    ToolCallPart,
    ToolCallPartDelta,
)
from pydantic_ai.models.test import TestModel
from pydantic_ai.run import AgentRunResultEvent

from ai.agent import AgentDeps, build_agent
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig


@dataclass(frozen=True)
class ReplWebServerConfig:
    host: str = "127.0.0.1"
    port: int = 8090


class ReplStreamRequest(BaseModel):
    question: str = Field(min_length=1, max_length=2000)
    canvas_id: str | None = Field(default=None, min_length=1, max_length=200)
    model: str = Field(default="test", min_length=1, max_length=200)
    base_url: str | None = None
    token: str | None = None
    org_id: str | None = None


def _normalize_optional(value: str | None) -> str | None:
    if value is None:
        return None
    normalized = value.strip()
    return normalized or None


def _resolve_required(value: str | None, env_name: str) -> str:
    resolved = _normalize_optional(value) or _normalize_optional(os.getenv(env_name))
    if resolved is None:
        raise ValueError(f"Missing required setting: {env_name}")
    return resolved


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
        return model_dump(mode="json")
    return str(value)


def _build_deps(payload: ReplStreamRequest) -> AgentDeps:
    if payload.canvas_id is None:
        raise ValueError("canvas_id is required for non-test models.")

    client = SuperplaneClient(
        SuperplaneClientConfig(
            base_url=_resolve_required(payload.base_url, "SUPERPLANE_BASE_URL"),
            api_token=_resolve_required(payload.token, "SUPERPLANE_API_TOKEN"),
            organization_id=_resolve_required(payload.org_id, "SUPERPLANE_ORG_ID"),
        )
    )
    return AgentDeps(
        client=client,
        default_canvas_id=payload.canvas_id,
        show_tool_calls=False,
        allow_canvas_details=False,
    )


async def _stream_agent_run(payload: ReplStreamRequest) -> AsyncIterator[dict[str, Any]]:
    started_at = time.perf_counter()
    yield {
        "type": "run_started",
        "model": payload.model,
        "canvas_id": payload.canvas_id,
    }

    if payload.model == "test":
        test_agent: Agent[None, str] = Agent(model=TestModel(), output_type=str)
        async for event in test_agent.run_stream_events(user_prompt=payload.question):
            if isinstance(event, PartDeltaEvent) and isinstance(event.delta, TextPartDelta):
                chunk = event.delta.content_delta
                if chunk:
                    yield {
                        "type": "model_delta",
                        "content": chunk,
                    }
                continue

            if isinstance(event, AgentRunResultEvent):
                yield {
                    "type": "final_answer",
                    "output": _to_jsonable(event.result.output),
                    "usage": _to_jsonable(event.result.usage()),
                }

        yield {
            "type": "run_completed",
            "elapsed_ms": (time.perf_counter() - started_at) * 1000,
        }
        yield {"type": "done"}
        return

    agent = build_agent(model=payload.model)
    deps = _build_deps(payload)
    deps.question_started_at = started_at
    deps.waiting_message_printed = False
    deps.allow_canvas_details = False

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
    async for event in agent.run_stream_events(user_prompt=payload.question, deps=deps):
        if isinstance(event, PartStartEvent) and isinstance(event.part, ToolCallPart):
            tool_call_id = event.part.tool_call_id
            if tool_call_id and likely_output_tool_name(event.part.tool_name):
                output_tool_call_id = tool_call_id
            continue

        if isinstance(event, FinalResultEvent):
            if event.tool_call_id:
                output_tool_call_id = event.tool_call_id
            continue

        if isinstance(event, PartDeltaEvent) and isinstance(event.delta, ToolCallPartDelta):
            tool_call_id = event.delta.tool_call_id
            if tool_call_id is None:
                continue

            # Some models stream output-tool JSON args incrementally.
            # Surface answer deltas as they become parseable.
            if output_tool_call_id is None and likely_output_tool_name(event.delta.tool_name_delta):
                output_tool_call_id = tool_call_id
            if output_tool_call_id != tool_call_id:
                continue

            args_delta = event.delta.args_delta
            if isinstance(args_delta, dict):
                maybe_delta = emit_answer_delta_from_output_args(tool_call_id, args_delta)
                if maybe_delta is not None:
                    yield maybe_delta
                continue

            if isinstance(args_delta, str):
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
                yield {
                    "type": "model_delta",
                    "content": chunk,
                }
            continue

        if isinstance(event, AgentRunResultEvent):
            result = event.result
            output = _to_jsonable(result.output)
            if isinstance(output, dict) and not streamed_any_answer_delta:
                answer = output.get("answer")
                if isinstance(answer, str) and answer:
                    for chunk in _iter_text_chunks(answer):
                        yield {
                            "type": "model_delta",
                            "content": chunk,
                        }
                        await asyncio.sleep(0.01)
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
    app = FastAPI()

    @app.post("/v1/repl/stream")
    async def stream_repl(payload: ReplStreamRequest, request: Request) -> StreamingResponse:
        async def event_generator() -> AsyncIterator[str]:
            try:
                async for event in _stream_agent_run(payload):
                    if await request.is_disconnected():
                        break
                    yield _encode_sse_event(event)
            except Exception as error:
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


class ReplWebServer:
    def __init__(self, config: ReplWebServerConfig) -> None:
        self._config = config
        self._app = _create_app()
        self._uvicorn_config = uvicorn.Config(
            self._app,
            host=config.host,
            port=config.port,
            log_level="warning",
            access_log=False,
        )
        self._server = uvicorn.Server(self._uvicorn_config)
        self._thread: threading.Thread | None = None

    @property
    def base_url(self) -> str:
        host = self._config.host
        if host == "0.0.0.0":
            host = "127.0.0.1"
        return f"http://{host}:{self._config.port}"

    def start(self) -> None:
        if self._thread is not None and self._thread.is_alive():
            return
        self._thread = threading.Thread(target=self.serve_forever, daemon=True)
        self._thread.start()
        for _ in range(200):
            if self._server.started:
                break
            time.sleep(0.01)

    def stop(self) -> None:
        self._server.should_exit = True
        if self._thread is not None:
            self._thread.join(timeout=5.0)

    def serve_forever(self) -> None:
        self._server.run()
