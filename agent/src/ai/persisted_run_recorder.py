from typing import Any

from pydantic_ai.messages import (
    FunctionToolResultEvent,
    ModelMessagesTypeAdapter,
    ModelRequest,
    ModelResponse,
    TextPart,
    ToolCallPart,
    UserPromptPart,
)

from ai.session_store import SessionStore


class PersistedRunRecorder:
    def __init__(self, store: SessionStore, chat_id: str, user_prompt: str) -> None:
        self._store = store
        self._chat_id = chat_id
        self._history_count_before_run = self._store.count_chat_model_messages(chat_id)
        self._authoritative_messages_saved = False
        self._current_response_message_id: str | None = None
        self._current_response: ModelResponse | None = None
        self._store.set_initial_chat_message_if_missing(chat_id, user_prompt)
        self._store.create_agent_chat_model_message(
            chat_id,
            ModelRequest(parts=[UserPromptPart(user_prompt)]),
        )

    @property
    def history_count_before_run(self) -> int:
        return self._history_count_before_run

    def _persist_current_response(self) -> None:
        if self._current_response is None or self._authoritative_messages_saved:
            return

        if self._current_response_message_id is None:
            record = self._store.create_agent_chat_model_message(self._chat_id, self._current_response)
            self._current_response_message_id = record.id
            return

        self._store.update_agent_chat_model_message(self._current_response_message_id, self._current_response)

    def save_authoritative_messages(self, messages: Any) -> None:
        validated_messages = ModelMessagesTypeAdapter.validate_python(messages)
        self._store.replace_agent_chat_messages_after(
            self._chat_id,
            self._history_count_before_run,
            list(validated_messages),
        )
        self._authoritative_messages_saved = True
        self._current_response_message_id = None
        self._current_response = None

    def append_assistant_content(self, chunk: str) -> None:
        if not chunk:
            return

        if self._current_response is None:
            self._current_response = ModelResponse(parts=[TextPart(chunk)])
            self._persist_current_response()
            return

        text_part_index = next(
            (index for index, part in enumerate(self._current_response.parts) if isinstance(part, TextPart)),
            None,
        )
        if text_part_index is None:
            self._current_response.parts = [*self._current_response.parts, TextPart(chunk)]
        else:
            text_part = self._current_response.parts[text_part_index]
            self._current_response.parts = [
                *self._current_response.parts[:text_part_index],
                TextPart(f"{text_part.content}{chunk}"),
                *self._current_response.parts[text_part_index + 1 :],
            ]

        self._persist_current_response()

    def set_assistant_content(self, content: str) -> None:
        self._current_response = ModelResponse(parts=[TextPart(content)])
        self._persist_current_response()

    def tool_started(self, part: ToolCallPart) -> None:
        if self._current_response is None:
            self._current_response = ModelResponse(parts=[part])
            self._persist_current_response()
            return

        updated_parts = [
            existing_part
            for existing_part in self._current_response.parts
            if not isinstance(existing_part, ToolCallPart) or existing_part.tool_call_id != part.tool_call_id
        ]
        updated_parts.append(part)
        self._current_response.parts = updated_parts
        self._persist_current_response()

    def tool_call_delta(self, tool_call_id: str, args_delta: str | dict[str, Any] | None, tool_name: str | None) -> None:
        if self._current_response is None:
            self._current_response = ModelResponse(parts=[])

        updated = False
        next_parts: list[Any] = []
        for part in self._current_response.parts:
            if not isinstance(part, ToolCallPart) or part.tool_call_id != tool_call_id:
                next_parts.append(part)
                continue

            next_tool_name = tool_name or part.tool_name
            next_args: str | dict[str, Any] | None = part.args
            if isinstance(args_delta, dict):
                next_args = args_delta
            elif isinstance(args_delta, str):
                if isinstance(next_args, str):
                    next_args = f"{next_args}{args_delta}"
                elif next_args is None:
                    next_args = args_delta
            next_parts.append(
                ToolCallPart(
                    tool_name=next_tool_name,
                    args=next_args,
                    tool_call_id=tool_call_id,
                )
            )
            updated = True

        if not updated:
            next_parts.append(
                ToolCallPart(
                    tool_name=tool_name or "tool",
                    args=args_delta if isinstance(args_delta, (str, dict)) else None,
                    tool_call_id=tool_call_id,
                )
            )

        self._current_response.parts = next_parts
        self._persist_current_response()

    def tool_finished(self, event: FunctionToolResultEvent) -> None:
        parts: list[Any] = [event.result]
        if event.content:
            parts.append(UserPromptPart(event.content))
        self._store.create_agent_chat_model_message(
            self._chat_id,
            ModelRequest(parts=parts),
        )
        self._current_response_message_id = None
        self._current_response = None
