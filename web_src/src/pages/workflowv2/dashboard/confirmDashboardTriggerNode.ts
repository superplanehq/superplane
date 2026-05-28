import type { DashboardContextValue } from "./DashboardContext";
import { DASHBOARD_TRIGGER_NODE_EVENT } from "./dashboardEvents";

/**
 * Fire the configured trigger after the user confirms in a console Run dialog.
 * Forwards the pre-built parameters object (already in the
 * `{ template, ...values }` shape that the backend expects) so the host
 * skips the default `mergeTriggerParameters` step it would normally apply
 * when no parameters are provided. Falls back to the legacy window event
 * when there is no live dashboard context (e.g. the dashboard rendered
 * outside the workflow page).
 */
export async function confirmDashboardTriggerNode(
  ctx: DashboardContextValue | undefined,
  nodeId: string,
  triggerName: string | undefined,
  parameters: Record<string, unknown>,
): Promise<void> {
  if (ctx?.onTriggerNode) {
    await ctx.onTriggerNode(nodeId, {
      hookName: "run",
      templateName: triggerName,
      parameters,
    });
    return;
  }
  window.dispatchEvent(
    new CustomEvent(DASHBOARD_TRIGGER_NODE_EVENT, {
      detail: { nodeId, triggerName, parameters },
    }),
  );
}
