import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onWorkflowCompletedTriggerRenderer } from "./on_workflow_completed";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./run_pipeline";
import { getWorkflowMapper } from "./get_workflow";
import { getLastWorkflowMapper } from "./get_last_workflow";
import { getRecentWorkflowRunsMapper } from "./get_recent_workflow_runs";
import { getTestMetricsMapper } from "./get_test_metrics";
import { getFlakyTestsMapper } from "./get_flaky_tests";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  runPipeline: runPipelineMapper,
  getWorkflow: getWorkflowMapper,
  getLastWorkflow: getLastWorkflowMapper,
  getRecentWorkflowRuns: getRecentWorkflowRunsMapper,
  getTestMetrics: getTestMetricsMapper,
  getFlakyTests: getFlakyTestsMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onWorkflowCompleted: onWorkflowCompletedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
};
