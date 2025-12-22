import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPipelineDoneTriggerRenderer } from "./on_pipeline_done";
import { RUN_WORKFLOW_STATE_REGISTRY, runWorkflowMapper } from "./run_workflow";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runWorkflow: runWorkflowMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineDone: onPipelineDoneTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runWorkflow: RUN_WORKFLOW_STATE_REGISTRY,
};