import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./create_issue";
import { getIssueMapper } from "./get_issue";
import { updateIssueMapper } from "./update_issue";
import { addIssueLabelMapper } from "./add_issue_label";
import { addIssueCommentMapper } from "./add_issue_comment";
import { onIssueTriggerRenderer } from "./on_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  addIssueLabel: addIssueLabelMapper,
  addIssueComment: addIssueCommentMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  addIssueLabel: buildActionStateRegistry("labeled"),
  addIssueComment: buildActionStateRegistry("commented"),
};
