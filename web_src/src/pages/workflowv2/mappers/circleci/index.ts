import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onWorkflowCompletedTriggerRenderer } from "./on_workflow_completed";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./run_pipeline";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runPipeline: runPipelineMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onWorkflowCompleted: onWorkflowCompletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
};
