import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { onPipelineDoneTriggerRenderer } from "./on_pipeline_done";
import { runWorkflowMapper } from "./run_workflow";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runWorkflow: runWorkflowMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineDone: onPipelineDoneTriggerRenderer,
};