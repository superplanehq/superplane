/** Canvas node as used for AI Builder @-mentions (transcoding to wire tokens). */
export type AiBuilderMentionNode = {
  id: string;
  name?: string;
  label?: string;
};

export function aiBuilderNodeDisplayName(node: AiBuilderMentionNode): string {
  const raw = (node.name ?? node.label ?? "").trim();
  return raw.length > 0 ? raw : "Untitled";
}

/**
 * Replace visible `@Canvas name` segments with `@[node:<id>]` for the agent.
 * Longest names first so shorter prefixes do not steal matches.
 */
export function transcodeAiNodeMentions(text: string, nodes: AiBuilderMentionNode[]): string {
  const pairs = nodes
    .map((n) => ({ id: n.id.trim(), name: aiBuilderNodeDisplayName(n) }))
    .filter((p) => p.id.length > 0 && p.name.length > 0);
  if (pairs.length === 0) {
    return text;
  }

  pairs.sort((a, b) => b.name.length - a.name.length);

  let out = text;
  for (const { id, name } of pairs) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const re = new RegExp(`(^|[\\s])@${escaped}(?=\\s|$|[.,!?;:])`, "g");
    out = out.replace(re, `$1@[node:${id}]`);
  }
  return out;
}
