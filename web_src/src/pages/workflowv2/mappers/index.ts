import { ComponentBaseMapper, TriggerRenderer, ComponentAdditionalDataBuilder } from "./types";
import { defaultTriggerRenderer } from "./default";
import { githubTriggerRenderer } from "./github";
import { scheduleTriggerRenderer } from "./schedule";
import { noopMapper } from "./noop";
import { ifMapper } from "./if";
import { httpMapper } from "./http";
import { semaphoreMapper } from "./semaphore";
import { timeGateMapper } from "./timegate";
import { filterMapper } from "./filter";
import { waitMapper } from "./wait";
import { approvalMapper, approvalDataBuilder } from "./approval";

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
  if: ifMapper,
  http: httpMapper,
  semaphore: semaphoreMapper,
  time_gate: timeGateMapper,
  filter: filterMapper,
  wait: waitMapper,
  approval: approvalMapper,
};

const componentAdditionalDataBuilders: Record<string, ComponentAdditionalDataBuilder> = {
  approval: approvalDataBuilder,
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
  return componentBaseMappers[componentName] || noopMapper;
}

/**
 * Get the appropriate additional data builder for a component type.
 * Returns undefined if no specific builder is registered.
 */
export function getComponentAdditionalDataBuilder(componentName: string): ComponentAdditionalDataBuilder | undefined {
  return componentAdditionalDataBuilders[componentName];
}
