import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";
import { onPullRequestTriggerRenderer } from "./on_pull_request";
import { onIssueTriggerRenderer } from "./on_issue";
import { onReleaseTriggerRenderer } from "./on_release";
import { baseActionMapper } from "./base";
import { runWorkflowMapper } from "./run_workflow";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: baseActionMapper,
  getIssue: baseActionMapper,
  updateIssue: baseActionMapper,
  createIssueComment: baseActionMapper,
  updateIssueComment: baseActionMapper,
  runWorkflow: runWorkflowMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
  onPullRequest: onPullRequestTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
};
