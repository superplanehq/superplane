import type { ConsoleContextValue } from "./ConsoleContext";
import { CONSOLE_TRIGGER_NODE_EVENT } from "./consoleEvents";

/**
 * Fire the configured trigger after the user confirms in a console Run dialog.
 * Forwards the pre-built parameters object (already in the
 * `{ template, ...values }` shape that the backend expects) so the host
 * skips the default `mergeTriggerParameters` step it would normally apply
 * when no parameters are provided. Falls back to the legacy window event
 * when there is no live dashboard context (e.g. the dashboard rendered
 * outside the workflow page).
 */
export async function confirmConsoleTriggerNode(
  ctx: ConsoleContextValue | undefined,
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
    new CustomEvent(CONSOLE_TRIGGER_NODE_EVENT, {
      detail: { nodeId, triggerName, parameters },
    }),
  );
}
