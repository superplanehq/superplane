import {
  ComponentBaseMapper,
  TriggerRenderer,
  ComponentAdditionalDataBuilder,
  EventStateRegistry,
  CustomFieldRenderer,
} from "./types";
import { defaultTriggerRenderer } from "./default";
import { githubTriggerRenderer } from "./github";
import { scheduleTriggerRenderer, scheduleCustomFieldRenderer } from "./schedule";
import { noopMapper } from "./noop";
import { ifMapper } from "./if";
import { httpMapper } from "./http";
import { semaphoreMapper as oldSemaphoreMapper } from "./semaphore";
import {
  componentMappers as semaphoreComponentMappers,
  triggerRenderers as semaphoreTriggerRenderers,
} from "./semaphore/index";
import { timeGateMapper } from "./timegate";
import { filterMapper } from "./filter";
import { waitMapper } from "./wait";
import { approvalMapper, approvalDataBuilder, APPROVAL_STATE_REGISTRY } from "./approval";
import { DEFAULT_STATE_REGISTRY } from "./stateRegistry";

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
  semaphore: oldSemaphoreMapper,
  time_gate: timeGateMapper,
  filter: filterMapper,
  wait: waitMapper,
  approval: approvalMapper,
};

const appMappers: Record<string, Record<string, ComponentBaseMapper>> = {
  semaphore: semaphoreComponentMappers,
};

const appTriggerRenderers: Record<string, Record<string, TriggerRenderer>> = {
  semaphore: semaphoreTriggerRenderers,
};

const componentAdditionalDataBuilders: Record<string, ComponentAdditionalDataBuilder> = {
  approval: approvalDataBuilder,
};

const eventStateRegistries: Record<string, EventStateRegistry> = {
  approval: APPROVAL_STATE_REGISTRY,
};

const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  schedule: scheduleCustomFieldRenderer,
};

/**
 * Get the appropriate renderer for a trigger type.
 * Falls back to the default renderer if no specific renderer is registered.
 */
export function getTriggerRenderer(name: string): TriggerRenderer {
  const parts = name.split(".");
  if (parts.length == 1) {
    return triggerRenderers[name] || defaultTriggerRenderer;
  }

  const appName = parts[0];
  const appTriggers = appTriggerRenderers[appName];
  if (!appTriggers) {
    return defaultTriggerRenderer;
  }

  const triggerName = parts[1];
  return appTriggers[triggerName] || defaultTriggerRenderer;
}

/**
 * Get the appropriate mapper for a component.
 * Falls back to the noop mapper if no specific mapper is registered.
 */
export function getComponentBaseMapper(name: string): ComponentBaseMapper {
  const parts = name.split(".");
  if (parts.length == 1) {
    return componentBaseMappers[name] || noopMapper;
  }

  const appName = parts[0];
  const appMapper = appMappers[appName];
  if (!appMapper) {
    return noopMapper;
  }

  const componentName = parts[1];
  return appMapper[componentName] || noopMapper;
}

/**
 * Get the appropriate additional data builder for a component type.
 * Returns undefined if no specific builder is registered.
 */
export function getComponentAdditionalDataBuilder(componentName: string): ComponentAdditionalDataBuilder | undefined {
  return componentAdditionalDataBuilders[componentName];
}

/**
 * Get the appropriate state registry for a component type.
 * Falls back to the default state registry if no specific registry is registered.
 */
export function getEventStateRegistry(componentName: string): EventStateRegistry {
  return eventStateRegistries[componentName] || DEFAULT_STATE_REGISTRY;
}

/**
 * Get the state map for a component type.
 * Falls back to the default state map if no specific registry is registered.
 */
export function getStateMap(componentName: string) {
  return getEventStateRegistry(componentName).stateMap;
}

/**
 * Get the state function for a component type.
 * Falls back to the default state function if no specific registry is registered.
 */
export function getState(componentName: string) {
  return getEventStateRegistry(componentName).getState;
}

/**
 * Get the appropriate custom field renderer for a component/trigger type.
 * Returns undefined if no specific renderer is registered.
 */
export function getCustomFieldRenderer(componentName: string): CustomFieldRenderer | undefined {
  return customFieldRenderers[componentName];
}
