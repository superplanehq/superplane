import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { acceptMergeRequestMapper } from "./accept_merge_request";
import { addReactionMapper } from "./add_reaction";
import { approveMergeRequestMapper } from "./approve_merge_request";
import { createDeploymentMapper } from "./create_deployment";
import { createDeploymentStatusMapper } from "./create_deployment_status";
import { createIssueMapper } from "./create_issue";
import { createMergeCommentMapper } from "./create_merge_comment";
import { onBranchCreatedTriggerRenderer } from "./on_branch_created";
import { onIssueTriggerRenderer } from "./on_issue";
import { onMergeCommentTriggerRenderer } from "./on_merge_comment";
import { onMergeRequestTriggerRenderer } from "./on_merge_request";
import { onMilestoneTriggerRenderer } from "./on_milestone";
import { onPipelineTriggerRenderer } from "./on_pipeline";
import { onPushTriggerRenderer } from "./on_push";
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
  createMergeComment: buildActionStateRegistry("created"),
  addReaction: buildActionStateRegistry("added"),
  acceptMergeRequest: buildActionStateRegistry("merged"),
  approveMergeRequest: buildActionStateRegistry("approved"),
  createDeployment: buildActionStateRegistry("created"),
  createDeploymentStatus: buildActionStateRegistry("updated"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  runPipeline: runPipelineMapper,
  getPipeline: pipelineLookupMapper,
  getLatestPipeline: pipelineLookupMapper,
  getTestReportSummary: testReportSummaryMapper,
  createMergeComment: createMergeCommentMapper,
  addReaction: addReactionMapper,
  acceptMergeRequest: acceptMergeRequestMapper,
  approveMergeRequest: approveMergeRequestMapper,
  createDeployment: createDeploymentMapper,
  createDeploymentStatus: createDeploymentStatusMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onBranchCreated: onBranchCreatedTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onMergeComment: onMergeCommentTriggerRenderer,
  onMergeRequest: onMergeRequestTriggerRenderer,
  onMilestone: onMilestoneTriggerRenderer,
  onPipeline: onPipelineTriggerRenderer,
  onPush: onPushTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTag: onTagTriggerRenderer,
  onVulnerability: onVulnerabilityTriggerRenderer,
};
