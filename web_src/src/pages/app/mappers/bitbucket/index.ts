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
  findPullRequest: findPullRequestMapper,
  createPullRequest: createPullRequestMapper,
  updatePullRequest: updatePullRequestMapper,
  createPullRequestComment: createPullRequestCommentMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  findPullRequest: FIND_PULL_REQUEST_STATE_REGISTRY,
  createPullRequest: buildActionStateRegistry("created"),
  updatePullRequest: buildActionStateRegistry("updated"),
  createPullRequestComment: buildActionStateRegistry("created"),
};
