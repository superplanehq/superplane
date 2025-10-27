import { TriggerRenderer } from "./types";
import { defaultTriggerRenderer } from "./default";
import { githubTriggerRenderer } from "./github";
import { scheduleTriggerRenderer } from "./schedule";

/**
 * Registry mapping trigger names to their renderers.
 *
 * To add a new trigger type with custom rendering:
 * 1. Create a new file in this directory (e.g., 'mytrigger.ts')
 * 2. Implement the TriggerRenderer interface
 * 3. Import it above and add it to this registry
 *
 * Any trigger type not in this registry will use the defaultTriggerRenderer.
 */
const triggerRenderers: Record<string, TriggerRenderer> = {
  github: githubTriggerRenderer,
  schedule: scheduleTriggerRenderer,
};

/**
 * Gets the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getTriggerRenderer(triggerName: string): TriggerRenderer {
  return triggerRenderers[triggerName] || defaultTriggerRenderer;
}
