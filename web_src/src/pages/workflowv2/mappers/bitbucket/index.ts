import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { baseIssueMapper } from "./base";
import { createIssueCommentMapper } from "./create_issue_comment";
import { onPushTriggerRenderer } from "./on_push";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  createIssueComment: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: baseIssueMapper,
  createIssueComment: createIssueCommentMapper,
  getIssue: baseIssueMapper,
  updateIssue: baseIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onPush: onPushTriggerRenderer,
};
