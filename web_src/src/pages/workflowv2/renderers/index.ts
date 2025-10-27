import { TriggerRenderer } from "./types";
import { defaultTriggerRenderer } from "./default";
import { githubTriggerRenderer } from "./github";
import { scheduleTriggerRenderer } from "./schedule";

/**
 * Registry mapping trigger names to their renderers.
 * Any trigger type not in this registry will use the defaultTriggerRenderer.
 */
const triggerRenderers: Record<string, TriggerRenderer> = {
  github: githubTriggerRenderer,
  schedule: scheduleTriggerRenderer,
};

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getTriggerRenderer(triggerName: string): TriggerRenderer {
  return triggerRenderers[triggerName] || defaultTriggerRenderer;
}
