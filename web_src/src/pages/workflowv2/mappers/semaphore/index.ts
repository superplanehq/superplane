import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { onPipelineDoneTriggerRenderer } from "./on_pipeline_done";
import { RUN_WORKFLOW_STATE_REGISTRY, runWorkflowMapper } from "./run_workflow";
import { getPipelineMapper } from "./get_pipeline";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runWorkflow: runWorkflowMapper,
  getPipeline: getPipelineMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineDone: onPipelineDoneTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runWorkflow: RUN_WORKFLOW_STATE_REGISTRY,
  getPipeline: buildActionStateRegistry("retrieved"),
};
