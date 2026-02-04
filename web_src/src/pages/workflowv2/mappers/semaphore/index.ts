import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPipelineDoneTriggerRenderer } from "./on_pipeline_done";
import { RUN_WORKFLOW_STATE_REGISTRY, runWorkflowMapper } from "./run_workflow";
import { getJobLogsMapper } from "./get_job_logs";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runWorkflow: runWorkflowMapper,
  getJobLogs: getJobLogsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPipelineDone: onPipelineDoneTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runWorkflow: RUN_WORKFLOW_STATE_REGISTRY,
};
