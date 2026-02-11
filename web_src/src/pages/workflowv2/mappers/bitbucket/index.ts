import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";

export const componentMappers: Record<string, ComponentBaseMapper> = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
};

export const customFieldRenderers: Record<string, never> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
