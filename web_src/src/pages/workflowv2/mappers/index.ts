import { ComponentBaseMapper, TriggerRenderer } from "./types";
import { defaultTriggerRenderer } from "./default";
import { githubTriggerRenderer } from "./github";
import { scheduleTriggerRenderer } from "./schedule";
import { noopMapper } from "./noop";
import { httpMapper } from "./http";

/**
 * Registry mapping trigger names to their renderers.
 * Any trigger type not in this registry will use the defaultTriggerRenderer.
 */
const triggerRenderers: Record<string, TriggerRenderer> = {
  github: githubTriggerRenderer,
  schedule: scheduleTriggerRenderer,
};

const componentBaseMappers: Record<string, ComponentBaseMapper> = {
  noop: noopMapper,
  http: httpMapper,
};

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getTriggerRenderer(triggerName: string): TriggerRenderer {
  return triggerRenderers[triggerName] || defaultTriggerRenderer;
}

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getComponentBaseMapper(componentName: string): ComponentBaseMapper {
  return componentBaseMappers[componentName];
}
