import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPipelineCompletedTriggerRenderer } from "./on_pipeline_completed";
import { TRIGGER_PIPELINE_STATE_REGISTRY, triggerPipelineMapper } from "./trigger_pipeline";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  triggerPipeline: triggerPipelineMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineCompleted: onPipelineCompletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  triggerPipeline: TRIGGER_PIPELINE_STATE_REGISTRY,
};
