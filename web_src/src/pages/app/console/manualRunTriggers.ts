import type { SuperplaneComponentsNode } from "@/api-client";

/**
 * Trigger registry identifiers (matching `node.component` for `TYPE_TRIGGER`
 * nodes) whose Run hook is user-invokable.
 *
 * Kept as a hardcoded frontend list on purpose: the backend registers only
 * `start` and `schedule` as user-runnable triggers (see
 * [pkg/triggers/start/start.go](../../../../pkg/triggers/start/start.go) and
 * [pkg/triggers/schedule/schedule.go](../../../../pkg/triggers/schedule/schedule.go));
 * every other built-in and integration trigger is event-driven. Adding a new
 * manual-run trigger already requires a UI PR (form copy, docs, hook plumbing),
 * so we skip the network round-trip and keep the list next to the components
 * that consume it. Backend authorization stays in `InvokeNodeTriggerHook`,
 * which still rejects non-user hooks server-side — this list only hides UI.
 */
export const MANUAL_RUN_TRIGGER_COMPONENTS: ReadonlySet<string> = new Set(["start", "schedule"]);

/**
 * Single source of truth for "this node can be manually run from the
 * console". True only when the node is a `TYPE_TRIGGER` and its component
 * appears in {@link MANUAL_RUN_TRIGGER_COMPONENTS}. Used by every widget that
 * exposes a Run affordance (node panel Run button, table row actions, editor
 * "Show Run" controls) so event triggers such as `github.onPullRequest` never
 * offer a manual-run affordance.
 */
export function isManualRunNode(node: SuperplaneComponentsNode | undefined): boolean {
  if (!node) return false;
  if (node.type !== "TYPE_TRIGGER") return false;
  return MANUAL_RUN_TRIGGER_COMPONENTS.has(node.component ?? "");
}
