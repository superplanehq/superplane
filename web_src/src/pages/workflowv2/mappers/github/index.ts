import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { onPushTriggerRenderer } from "./on_push";
import { onPullRequestTriggerRenderer } from "./on_pull_request";
import { onIssueTriggerRenderer } from "./on_issue";
import { onReleaseTriggerRenderer } from "./on_release";

export const componentMappers: Record<string, ComponentBaseMapper> = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
  onPullRequest: onPullRequestTriggerRenderer,
  onIssue: onIssueTriggerRenderer,
  onRelease: onReleaseTriggerRenderer,
};
