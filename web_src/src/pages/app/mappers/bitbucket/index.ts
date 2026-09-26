import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { getCommitStatusMapper, publishCommitStatusMapper } from "./commit_status_actions";
import { onCommitStatusTriggerRenderer } from "./on_commit_status";
import { onPRCommentTriggerRenderer } from "./on_pr_comment";
import { onPullRequestTriggerRenderer } from "./on_pull_request";
import { onPushTriggerRenderer } from "./on_push";
import {
  createPRCommentMapper,
  createPullRequestMapper,
  mergePullRequestMapper,
  updatePullRequestMapper,
} from "./pull_request_actions";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createPullRequest: buildActionStateRegistry("created"),
  updatePullRequest: buildActionStateRegistry("updated"),
  mergePullRequest: buildActionStateRegistry("merged"),
  createPRComment: buildActionStateRegistry("created"),
  publishCommitStatus: buildActionStateRegistry("published"),
  getCommitStatus: buildActionStateRegistry("retrieved"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createPullRequest: createPullRequestMapper,
  updatePullRequest: updatePullRequestMapper,
  mergePullRequest: mergePullRequestMapper,
  createPRComment: createPRCommentMapper,
  publishCommitStatus: publishCommitStatusMapper,
  getCommitStatus: getCommitStatusMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
  onPullRequest: onPullRequestTriggerRenderer,
  onPRComment: onPRCommentTriggerRenderer,
  onCommitStatus: onCommitStatusTriggerRenderer,
};
