from __future__ import annotations

import re
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ai.agent import AgentDeps

# Client sends tokens like @[node:some-node-id]
_NODE_TOKEN_RE = re.compile(r"@\[node:([^\]]+)\]")

# Must match expand_node_mentions_in_prompt (stripped before this marker for UI list API).
REFERENCED_NODES_APPENDIX_MARKER = "\n\n### Referenced nodes"


def strip_referenced_nodes_appendix_for_display(text: str) -> str:
    """Strip mention appendix for UI; stored model messages still keep the full text."""
    if not text.strip():
        return text
    idx = text.find(REFERENCED_NODES_APPENDIX_MARKER)
    if idx == -1:
        return text
    return text[:idx].rstrip()


def parse_node_mention_ids(question: str) -> list[str]:
    return list(dict.fromkeys(_NODE_TOKEN_RE.findall(question)))


def expand_node_mentions_in_prompt(question: str, deps: AgentDeps) -> str:
    """Append a short appendix for @[node:...] tokens; prime deps.canvas_cache."""
    ids = parse_node_mention_ids(question)
    if not ids:
        return question

    canvas_id = deps.canvas_id
    summary = deps.canvas_cache.get(canvas_id)
    if summary is None:
        summary = deps.client.describe_canvas(canvas_id)
        deps.canvas_cache[canvas_id] = summary

    by_id = {node.id: node for node in summary.nodes}
    bullet_lines: list[str] = []
    for node_id in ids:
        node = by_id.get(node_id)
        if node is None:
            bullet_lines.append(f"- id `{node_id}`: (not found on canvas)")
            continue
        label = node.name or node.id
        bullet_lines.append(
            f"- **{label}** (`{node.id}`): "
            f"type={node.type or 'n/a'}, block={node.block_name or 'n/a'}"
        )

    return f"{question.rstrip()}{REFERENCED_NODES_APPENDIX_MARKER}\n\n" + "\n".join(bullet_lines)
