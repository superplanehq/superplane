import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { onBuildFinishedCustomFieldRenderer, onBuildFinishedTriggerRenderer } from "./on_build_finished";
import { TRIGGER_BUILD_STATE_REGISTRY, triggerBuildMapper } from "./trigger_build";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerBuild: triggerBuildMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onBuildFinished: onBuildFinishedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerBuild: TRIGGER_BUILD_STATE_REGISTRY,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onBuildFinished: onBuildFinishedCustomFieldRenderer,
};
