import { ComponentBaseMapper, CustomFieldRenderer, EventStateRegistry, TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";
import { onPullRequestTriggerRenderer } from "./on_pull_request";
import { onIssueTriggerRenderer } from "./on_issue";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";
import { onReleaseTriggerRenderer } from "./on_release";
import { onTagCreatedTriggerRenderer } from "./on_tag_created";
import { onBranchCreatedTriggerRenderer } from "./on_branch_created";
import { onPullRequestReviewCommentTriggerRenderer } from "./on_pr_review_comment";
import { onWorkflowRunTriggerRenderer } from "./on_workflow_run";
import { baseIssueMapper } from "./base";
import { RUN_WORKFLOW_STATE_REGISTRY, runWorkflowMapper, runWorkflowCustomFieldRenderer } from "./run_workflow";
import { publishCommitStatusMapper } from "./publish_commit_status";
import { createIssueCommentMapper } from "./create_issue_comment";
import { createReleaseMapper } from "./create_release";
import { updateReleaseMapper } from "./update_release";
import { deleteReleaseMapper } from "./delete_release";
import { getReleaseMapper } from "./get_release";
import { createReviewMapper } from "./create_review";
import { getWorkflowUsageMapper } from "./get_workflow_usage";
import { buildActionStateRegistry } from "../utils";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runWorkflow: RUN_WORKFLOW_STATE_REGISTRY,
  createIssue: buildActionStateRegistry("created"),
  createIssueComment: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  createReview: buildActionStateRegistry("created"),
  publishCommitStatus: buildActionStateRegistry("published"),
  createRelease: buildActionStateRegistry("created"),
  updateRelease: buildActionStateRegistry("updated"),
  deleteRelease: buildActionStateRegistry("deleted"),
  getRelease: buildActionStateRegistry("retrieved"),
  getWorkflowUsage: buildActionStateRegistry("retrieved"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: baseIssueMapper,
  createIssueComment: createIssueCommentMapper,
  getIssue: baseIssueMapper,
  updateIssue: baseIssueMapper,
  createReview: createReviewMapper,
  runWorkflow: runWorkflowMapper,
  publishCommitStatus: publishCommitStatusMapper,
  createRelease: createReleaseMapper,
  updateRelease: updateReleaseMapper,
  deleteRelease: deleteReleaseMapper,
  getRelease: getReleaseMapper,
  getWorkflowUsage: getWorkflowUsageMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
  onPullRequest: onPullRequestTriggerRenderer,
  onPRComment: onPullRequestReviewCommentTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onIssueComment: onIssueCommentTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTagCreated: onTagCreatedTriggerRenderer,
  onBranchCreated: onBranchCreatedTriggerRenderer,
  onWorkflowRun: onWorkflowRunTriggerRenderer,
};

export const customFieldRenderers: Record<string, CustomFieldRenderer> = {
  runWorkflow: runWorkflowCustomFieldRenderer,
};
