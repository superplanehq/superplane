from __future__ import annotations

import re
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ai.agent import AgentDeps

# Client sends tokens like @[node:some-node-id]
_NODE_TOKEN_RE = re.compile(r"@\[node:([^\]]+)\]")


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
    lines = ["### Referenced nodes", ""]
    for node_id in ids:
        node = by_id.get(node_id)
        if node is None:
            lines.append(f"- id `{node_id}`: (not found on canvas)")
            continue
        label = node.name or node.id
        lines.append(
            f"- **{label}** (`{node.id}`): type={node.type or 'n/a'}, block={node.block_name or 'n/a'}"
        )

    return f"{question.rstrip()}\n\n" + "\n".join(lines)
