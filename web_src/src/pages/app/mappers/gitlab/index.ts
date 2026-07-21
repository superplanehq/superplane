import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { acceptMergeRequestMapper } from "./accept_merge_request";
import { addMergeRequestReviewersMapper } from "./add_merge_request_reviewers";
import { addReactionMapper } from "./add_reaction";
import { approveMergeRequestMapper } from "./approve_merge_request";
import { createMergeRequestMapper } from "./create_merge_request";
import { removeMergeRequestReviewersMapper } from "./remove_merge_request_reviewers";
import { createDeploymentMapper } from "./create_deployment";
import { createDeploymentStatusMapper } from "./create_deployment_status";
import { createIssueMapper } from "./create_issue";
import { createIssueCommentMapper } from "./create_issue_comment";
import { createMergeCommentMapper } from "./create_merge_comment";
import { getIssueMapper } from "./get_issue";
import { onBranchCreatedTriggerRenderer } from "./on_branch_created";
import { onIssueTriggerRenderer } from "./on_issue";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";
import { onMergeCommentTriggerRenderer } from "./on_merge_comment";
import { onMergeRequestTriggerRenderer } from "./on_merge_request";
import { onMilestoneTriggerRenderer } from "./on_milestone";
import { onMRDiffNoteTriggerRenderer } from "./on_mr_diff_note";
import { onPipelineTriggerRenderer } from "./on_pipeline";
import { onPushTriggerRenderer } from "./on_push";
import { onReleaseTriggerRenderer } from "./on_release";
import { onTagTriggerRenderer } from "./on_tag";
import { onVulnerabilityTriggerRenderer } from "./on_vulnerability";
import { RUN_PIPELINE_STATE_REGISTRY, runPipelineMapper } from "./run_pipeline";
import { pipelineLookupMapper, testReportSummaryMapper } from "./pipeline_actions";
import { updateIssueMapper } from "./update_issue";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  runPipeline: RUN_PIPELINE_STATE_REGISTRY,
  getPipeline: buildActionStateRegistry("retrieved"),
  getLatestPipeline: buildActionStateRegistry("retrieved"),
  getTestReportSummary: buildActionStateRegistry("retrieved"),
  createMergeComment: buildActionStateRegistry("created"),
  addReaction: buildActionStateRegistry("added"),
  createMergeRequest: buildActionStateRegistry("created"),
  addMergeRequestReviewers: buildActionStateRegistry("updated"),
  removeMergeRequestReviewers: buildActionStateRegistry("updated"),
  acceptMergeRequest: buildActionStateRegistry("merged"),
  approveMergeRequest: buildActionStateRegistry("approved"),
  createDeployment: buildActionStateRegistry("created"),
  createDeploymentStatus: buildActionStateRegistry("updated"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  createIssueComment: buildActionStateRegistry("created"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  runPipeline: runPipelineMapper,
  getPipeline: pipelineLookupMapper,
  getLatestPipeline: pipelineLookupMapper,
  getTestReportSummary: testReportSummaryMapper,
  createMergeComment: createMergeCommentMapper,
  addReaction: addReactionMapper,
  createMergeRequest: createMergeRequestMapper,
  addMergeRequestReviewers: addMergeRequestReviewersMapper,
  removeMergeRequestReviewers: removeMergeRequestReviewersMapper,
  acceptMergeRequest: acceptMergeRequestMapper,
  approveMergeRequest: approveMergeRequestMapper,
  createDeployment: createDeploymentMapper,
  createDeploymentStatus: createDeploymentStatusMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  createIssueComment: createIssueCommentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onBranchCreated: onBranchCreatedTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onIssueComment: onIssueCommentTriggerRenderer,
  onMergeComment: onMergeCommentTriggerRenderer,
  onMergeRequest: onMergeRequestTriggerRenderer,
  onMilestone: onMilestoneTriggerRenderer,
  onMRDiffNote: onMRDiffNoteTriggerRenderer,
  onPipeline: onPipelineTriggerRenderer,
  onPush: onPushTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTag: onTagTriggerRenderer,
  onVulnerability: onVulnerabilityTriggerRenderer,
};
