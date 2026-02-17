import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onBuildFinishedTriggerRenderer } from "./on_build_finished";
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
