/**
 * Typed content shape and validator for the plural `nodes` dashboard panel.
 *
 * Lives in its own module so the main `panelTypes.ts` stays under the lint
 * budget; both the template seed and the validator are re-exported from
 * there so callers keep importing through a single entry point.
 *
 * Keep this in lockstep with `validateNodesPanelContent` in
 * `pkg/models/canvas_dashboard_yml.go`.
 */

/**
 * One entry in a {@link NodesPanelContent} list. The minimum required field
 * is `node` (canvas node id or name); the optional `label` overrides the
 * resolved node name in the rendered row, and `description` shows a short
 * supporting line (the node's purpose in this workflow).
 */
export interface NodesPanelNode {
  /** Canvas node id or name. Required. */
  node: string;
  /** Optional override for the row label. Falls back to the resolved node name. */
  label?: string;
  /** Optional short purpose line rendered under the node label. */
  description?: string;
  /** When true and the viewer has run permission, render a manual-run button. */
  showRun?: boolean;
  /** Optional override for the trigger template name (for nodes with multiple triggers). */
  triggerName?: string;
}

export interface NodesPanelContent {
  title?: string;
  /**
   * Configured nodes to render in this panel. A newly added panel may have
   * an empty array; the card body renders a "configure me" hint until the
   * author adds at least one entry through the form.
   */
  nodes: NodesPanelNode[];
}

/** Default content for a newly added `nodes` panel. */
export function templateForNodesPanel(defaultTitle?: string): NodesPanelContent {
  return { title: defaultTitle ?? "", nodes: [] };
}

/** Validate the persisted `nodes` content. Returns null when valid. */
export function validateNodesContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  if (obj.title !== undefined && obj.title !== null && typeof obj.title !== "string") {
    return "content.title must be a string.";
  }
  if (!Array.isArray(obj.nodes)) {
    return "content.nodes must be an array.";
  }
  for (let i = 0; i < obj.nodes.length; i += 1) {
    const error = validateNodesEntry(obj.nodes[i], i);
    if (error) return error;
  }
  return null;
}

function validateNodesEntry(raw: unknown, index: number): string | null {
  const entry = asObject(raw);
  if (!entry) return `content.nodes[${index}] must be an object.`;
  if (typeof entry.node !== "string" || entry.node.trim() === "") {
    return `content.nodes[${index}].node must be a non-empty string (canvas node id or name).`;
  }
  if (entry.label !== undefined && entry.label !== null && typeof entry.label !== "string") {
    return `content.nodes[${index}].label must be a string.`;
  }
  if (entry.description !== undefined && entry.description !== null && typeof entry.description !== "string") {
    return `content.nodes[${index}].description must be a string.`;
  }
  if (entry.showRun !== undefined && typeof entry.showRun !== "boolean") {
    return `content.nodes[${index}].showRun must be a boolean.`;
  }
  if (entry.triggerName !== undefined && entry.triggerName !== null && typeof entry.triggerName !== "string") {
    return `content.nodes[${index}].triggerName must be a string.`;
  }
  return null;
}

function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}
