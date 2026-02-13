import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./create_issue";
import { onIssueTriggerRenderer } from "./on_issue";
import { onMergeRequestTriggerRenderer } from "./on_merge_request";
import { onMilestoneTriggerRenderer } from "./on_milestone";
import { onPipelineTriggerRenderer } from "./on_pipeline";
import { onReleaseTriggerRenderer } from "./on_release";
import { onTagTriggerRenderer } from "./on_tag";
import { onVulnerabilityTriggerRenderer } from "./on_vulnerability";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./run_pipeline";
import { pipelineLookupMapper, testReportSummaryMapper } from "./pipeline_actions";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
  getPipeline: buildActionStateRegistry("retrieved"),
  getLatestPipeline: buildActionStateRegistry("retrieved"),
  getTestReportSummary: buildActionStateRegistry("retrieved"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  runPipeline: runPipelineMapper,
  getPipeline: pipelineLookupMapper,
  getLatestPipeline: pipelineLookupMapper,
  getTestReportSummary: testReportSummaryMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
  onMergeRequest: onMergeRequestTriggerRenderer,
  onMilestone: onMilestoneTriggerRenderer,
  onPipeline: onPipelineTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTag: onTagTriggerRenderer,
  onVulnerability: onVulnerabilityTriggerRenderer,
};
