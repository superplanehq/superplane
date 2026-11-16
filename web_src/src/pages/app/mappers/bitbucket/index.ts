import { reviewCandidatesMapper, ensureReviewersMapper, ENSURE_REVIEWERS_STATE_REGISTRY } from "./reviewers";
import { updatePullRequestBodySectionMapper } from "../updatePullRequestBodySection";
import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../eventDisplay";
import { onPushTriggerRenderer } from "./on_push";
import {
  FIND_PULL_REQUEST_STATE_REGISTRY,
  createPullRequestCommentMapper,
  createPullRequestMapper,
  findPullRequestMapper,
  updatePullRequestMapper,
} from "./pull_requests";

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  listPullRequestReviewCandidates: reviewCandidatesMapper,
  ensurePullRequestReviewers: ensureReviewersMapper,
  findPullRequest: findPullRequestMapper,
  createPullRequest: createPullRequestMapper,
  updatePullRequest: updatePullRequestMapper,
  updatePullRequestBodySection: updatePullRequestBodySectionMapper,
  createPullRequestComment: createPullRequestCommentMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  listPullRequestReviewCandidates: buildActionStateRegistry("listed"),
  ensurePullRequestReviewers: ENSURE_REVIEWERS_STATE_REGISTRY,
  findPullRequest: FIND_PULL_REQUEST_STATE_REGISTRY,
  createPullRequest: buildActionStateRegistry("created"),
  updatePullRequest: buildActionStateRegistry("updated"),
  createPullRequestComment: buildActionStateRegistry("created"),
};
