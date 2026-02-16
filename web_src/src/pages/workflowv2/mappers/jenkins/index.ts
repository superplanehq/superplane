import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { TRIGGER_BUILD_STATE_REGISTRY, triggerBuildCustomFieldRenderer, triggerBuildMapper } from "./trigger_build";
import { onBuildFinishedTriggerRenderer } from "./on_build_finished";

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
  triggerBuild: triggerBuildCustomFieldRenderer,
};
