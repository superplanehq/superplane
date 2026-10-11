import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../eventDisplay";
import { createIssueMapper } from "./create_issue";
import { deleteIssueMapper } from "./delete_issue";
import { getIssueMapper } from "./get_issue";
import { updateIssueMapper } from "./update_issue";
import { getWorkflowMapper } from "./get_workflow";
import { transitionIssueMapper } from "./transition_issue";
import { onIssueTriggerRenderer } from "./on_issue";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  deleteIssue: deleteIssueMapper,
  getWorkflow: getWorkflowMapper,
  transitionIssue: transitionIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
  onIssueComment: onIssueCommentTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  deleteIssue: buildActionStateRegistry("deleted"),
  getWorkflow: buildActionStateRegistry("retrieved"),
  transitionIssue: buildActionStateRegistry("transitioned"),
  onIssue: buildActionStateRegistry("triggered"),
  onIssueComment: buildActionStateRegistry("triggered"),
};
