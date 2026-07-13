import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { acceptMergeRequestMapper } from "./accept_merge_request";
import { addReactionMapper } from "./add_reaction";
import { approveMergeRequestMapper } from "./approve_merge_request";
import { createIssueMapper } from "./create_issue";
import { createMergeCommentMapper } from "./create_merge_comment";
import { getIssueMapper } from "./get_issue";
import { onIssueTriggerRenderer } from "./on_issue";
import { onMergeCommentTriggerRenderer } from "./on_merge_comment";
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
  getIssue: buildActionStateRegistry("retrieved"),
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
  getPipeline: buildActionStateRegistry("retrieved"),
  getLatestPipeline: buildActionStateRegistry("retrieved"),
  getTestReportSummary: buildActionStateRegistry("retrieved"),
  createMergeComment: buildActionStateRegistry("created"),
  addReaction: buildActionStateRegistry("added"),
  acceptMergeRequest: buildActionStateRegistry("merged"),
  approveMergeRequest: buildActionStateRegistry("approved"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  runPipeline: runPipelineMapper,
  getPipeline: pipelineLookupMapper,
  getLatestPipeline: pipelineLookupMapper,
  getTestReportSummary: testReportSummaryMapper,
  createMergeComment: createMergeCommentMapper,
  addReaction: addReactionMapper,
  acceptMergeRequest: acceptMergeRequestMapper,
  approveMergeRequest: approveMergeRequestMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
  onMergeComment: onMergeCommentTriggerRenderer,
  onMergeRequest: onMergeRequestTriggerRenderer,
  onMilestone: onMilestoneTriggerRenderer,
  onPipeline: onPipelineTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTag: onTagTriggerRenderer,
  onVulnerability: onVulnerabilityTriggerRenderer,
};
