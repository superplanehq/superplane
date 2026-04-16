import json
from dataclasses import dataclass, field
from typing import Any


_DISCOVERY_TOOL_NAMES = {
    "describe_component",
    "describe_trigger",
    "list_available_integrations",
    "list_components",
    "list_decision_patterns",
    "list_integration_resources",
    "list_org_integrations",
    "list_triggers",
    "search_decision_patterns",
}


def _normalize_tool_args(value: Any, *, depth: int = 0) -> Any:
    if depth > 4:
        return "<max-depth>"
    if value is None or isinstance(value, (bool, int, float)):
        return value
    if isinstance(value, str):
        return " ".join(value.split())
    if isinstance(value, dict):
        return {
            str(key): _normalize_tool_args(item, depth=depth + 1)
            for key, item in sorted(value.items(), key=lambda entry: str(entry[0]))
        }
    if isinstance(value, (list, tuple)):
        return [_normalize_tool_args(item, depth=depth + 1) for item in value[:20]]
    return str(value)


def is_guarded_tool(tool_name: str | None) -> bool:
    if not isinstance(tool_name, str):
        return False
    normalized = tool_name.strip().lower()
    if not normalized:
        return False
    if normalized in _DISCOVERY_TOOL_NAMES:
        return True
    return normalized.startswith("list_")


def build_tool_signature(tool_name: str | None, args: Any) -> str:
    normalized_tool_name = tool_name.strip().lower() if isinstance(tool_name, str) else "tool"
    return json.dumps(
        {
            "tool": normalized_tool_name,
            "args": _normalize_tool_args(args),
        },
        ensure_ascii=True,
        separators=(",", ":"),
        sort_keys=True,
    )


def is_no_progress_tool_result(content: Any) -> bool:
    if isinstance(content, dict):
        return bool(content.get("__tool_error__")) or bool(content.get("__tool_empty__"))

    if isinstance(content, list):
        if not content:
            return True
        if all(
            isinstance(item, dict)
            and (item.get("__tool_error__") or item.get("__tool_empty__"))
            for item in content
        ):
            return True

    return False


def describe_no_progress_result(content: Any) -> str:
    entry: dict[str, Any] | None = None
    if isinstance(content, dict):
        entry = content
    elif isinstance(content, list) and content and isinstance(content[0], dict):
        entry = content[0]

    if not entry:
        return "The agent repeated a discovery tool call without making progress."

    error_code = entry.get("__tool_error_code__")
    if error_code == "missing_integration_id":
        return (
            "I stopped because I kept retrying a discovery tool without an integration ID. "
            "Tell me which integration you want me to inspect and I can continue."
        )
    if error_code == "missing_resource_type":
        return (
            "I stopped because I kept retrying a discovery tool without a resource type. "
            "Tell me the exact resource type you want from that integration and I can continue."
        )
    if entry.get("__tool_empty__"):
        return (
            "I stopped because I kept repeating the same discovery step without finding any "
            "matching resources. Please confirm the integration and exact resource type to inspect."
        )
    return (
        "I stopped because I kept repeating the same discovery step without making progress. "
        "Please narrow the integration or resource you want me to inspect."
    )


@dataclass(slots=True)
class ToolLoopDecision:
    tool_name: str
    signature: str
    repeated_count: int
    message: str


@dataclass(slots=True)
class ToolLoopGuard:
    max_repeated_no_progress: int = 3
    pending_signatures_by_call_id: dict[str, str] = field(default_factory=dict)
    last_signature: str | None = None
    last_tool_name: str | None = None
    consecutive_no_progress_count: int = 0

    def register_call(self, tool_call_id: str, tool_name: str | None, args: Any) -> None:
        if not tool_call_id or not is_guarded_tool(tool_name):
            return
        self.pending_signatures_by_call_id[tool_call_id] = build_tool_signature(tool_name, args)

    def observe_result(
        self, tool_call_id: str, tool_name: str | None, content: Any
    ) -> ToolLoopDecision | None:
        signature = self.pending_signatures_by_call_id.pop(tool_call_id, None)
        if signature is None:
            return None

        if not is_no_progress_tool_result(content):
            self.reset()
            return None

        normalized_tool_name = tool_name.strip().lower() if isinstance(tool_name, str) else "tool"
        if signature == self.last_signature and normalized_tool_name == self.last_tool_name:
            self.consecutive_no_progress_count += 1
        else:
            self.last_signature = signature
            self.last_tool_name = normalized_tool_name
            self.consecutive_no_progress_count = 1

        if self.consecutive_no_progress_count < self.max_repeated_no_progress:
            return None

        return ToolLoopDecision(
            tool_name=normalized_tool_name,
            signature=signature,
            repeated_count=self.consecutive_no_progress_count,
            message=describe_no_progress_result(content),
        )

    def reset(self) -> None:
        self.last_signature = None
        self.last_tool_name = None
        self.consecutive_no_progress_count = 0
