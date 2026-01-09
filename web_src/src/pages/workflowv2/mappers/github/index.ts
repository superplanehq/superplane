import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";
import { onPullRequestTriggerRenderer } from "./on_pull_request";
import { onIssueTriggerRenderer } from "./on_issue";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";
import { onReleaseTriggerRenderer } from "./on_release";
import { onTagCreatedTriggerRenderer } from "./on_tag_created";
import { onBranchCreatedTriggerRenderer } from "./on_branch_created";
import { baseIssueMapper } from "./base";
import { RUN_WORKFLOW_STATE_REGISTRY, runWorkflowMapper } from "./run_workflow";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  runWorkflow: RUN_WORKFLOW_STATE_REGISTRY,
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: baseIssueMapper,
  getIssue: baseIssueMapper,
  updateIssue: baseIssueMapper,
  runWorkflow: runWorkflowMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
  onPullRequest: onPullRequestTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onIssueComment: onIssueCommentTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
  onTagCreated: onTagCreatedTriggerRenderer,
  onBranchCreated: onBranchCreatedTriggerRenderer,
};
