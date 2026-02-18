import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { onPipelineCompletedTriggerRenderer, onPipelineCompletedCustomFieldRenderer } from "./on_pipeline_completed";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./run_pipeline";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runPipeline: runPipelineMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineCompleted: onPipelineCompletedTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  onPipelineCompleted: onPipelineCompletedCustomFieldRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
};
