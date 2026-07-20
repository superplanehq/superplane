/**
 * Typed content shape and validator for the plural `nodes` dashboard panel.
 *
 * Lives in its own module so the main `panelTypes.ts` stays under the lint
 * budget; both the template seed and the validator are re-exported from
 * there so callers keep importing through a single entry point.
 *
 * Keep this in lockstep with `validateNodesPanelContent` in
 * `pkg/yaml/console.go`.
 */

import { asObject, optionalBooleanError, optionalStringError } from "./panelContentValidation";

/** Accepted values for `NodesPanelNode.formMode`. */
export const NODES_PANEL_FORM_MODES = ["modal", "inline"] as const;
export type NodesPanelFormMode = (typeof NODES_PANEL_FORM_MODES)[number];

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
  /**
   * When true, clicking Run always opens the confirm dialog — even for
   * templates with no input fields. When false (default), a parameter-less
   * template fires immediately; templates with input fields always prompt.
   */
  promptConfirmation?: boolean;
  /**
   * How the run parameter form is presented. Default `"modal"` reproduces
   * today's behavior (Run button opens {@link NodeRunConfirmDialog}).
   * `"inline"` renders {@link StartRunParameterFields} plus a submit button
   * directly in the panel body for prompt-submission style widgets. Only
   * honored when the entry resolves to a manual-run Start trigger whose
   * selected template exposes at least one parameter; otherwise the entry
   * falls back to the modal path.
   */
  formMode?: NodesPanelFormMode;
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

/**
 * Fold a legacy single-node panel body into a one-entry `nodes` panel. Used
 * by the panel router so both the `node` and `nodes` panel types share the
 * merged renderer, and by the panel state on first edit so the persisted
 * panel migrates to `type: nodes` (see `useConsolePanelState`).
 *
 * The reverse (unfolding a one-entry `nodes` panel back into a `node`) is
 * intentionally not provided: the compact single-node layout falls out of
 * the `nodes.length === 1` render branch, so both shapes look identical
 * once merged.
 */
export function nodesPanelContentFromLegacyNode(raw: Record<string, unknown> | undefined): NodesPanelContent {
  const obj = asObject(raw) ?? {};
  const node = typeof obj.node === "string" ? obj.node : "";
  const entry: NodesPanelNode = {
    node,
    label: typeof obj.label === "string" ? obj.label : undefined,
    showRun: typeof obj.showRun === "boolean" ? obj.showRun : false,
    triggerName: typeof obj.triggerName === "string" ? obj.triggerName : undefined,
    promptConfirmation: typeof obj.promptConfirmation === "boolean" ? obj.promptConfirmation : false,
  };
  // Always fold into exactly one entry — even when the legacy node is
  // unset — so the merged renderer keeps the compact single-node layout
  // and its "pick a node" empty state, matching the pre-merge card.
  return {
    title: typeof obj.title === "string" ? obj.title : "",
    nodes: [entry],
  };
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
  const prefix = `content.nodes[${index}]`;
  return (
    optionalStringError(`${prefix}.label`, entry.label) ??
    optionalStringError(`${prefix}.description`, entry.description) ??
    optionalBooleanError(`${prefix}.showRun`, entry.showRun) ??
    optionalStringError(`${prefix}.triggerName`, entry.triggerName) ??
    optionalBooleanError(`${prefix}.promptConfirmation`, entry.promptConfirmation) ??
    optionalFormModeError(`${prefix}.formMode`, entry.formMode)
  );
}

function optionalFormModeError(path: string, value: unknown): string | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "string" || !NODES_PANEL_FORM_MODES.includes(value as NodesPanelFormMode)) {
    return `${path} must be one of ${NODES_PANEL_FORM_MODES.map((m) => JSON.stringify(m)).join(", ")}.`;
  }
  return null;
}
