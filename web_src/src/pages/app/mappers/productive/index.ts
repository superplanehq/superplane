import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onTaskTriggerRenderer } from "./on_task";

export const componentMappers: Record<string, ComponentBaseMapper> = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onTask: onTaskTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
