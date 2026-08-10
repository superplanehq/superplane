import type { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./create_issue";
import { getIssueMapper } from "./get_issue";
import { updateIssueMapper } from "./update_issue";
import { addIssueLabelMapper } from "./add_issue_label";
import { addIssueCommentMapper } from "./add_issue_comment";
import { updateIssueCommentMapper } from "./update_issue_comment";
import { createAttachmentMapper } from "./create_attachment";
import { deleteAttachmentMapper } from "./delete_attachment";
import { removeIssueLabelMapper } from "./remove_issue_label";
import { addReactionMapper } from "./add_reaction";
import { onIssueTriggerRenderer } from "./on_issue";
import { onIssueLabelTriggerRenderer } from "./on_issue_label";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";
import { onIssueAttachmentTriggerRenderer } from "./on_issue_attachment";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
  getIssue: getIssueMapper,
  updateIssue: updateIssueMapper,
  addIssueLabel: addIssueLabelMapper,
  addIssueComment: addIssueCommentMapper,
  updateIssueComment: updateIssueCommentMapper,
  createAttachment: createAttachmentMapper,
  deleteAttachment: deleteAttachmentMapper,
  removeIssueLabel: removeIssueLabelMapper,
  addReaction: addReactionMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
  onIssueLabel: onIssueLabelTriggerRenderer,
  onIssueComment: onIssueCommentTriggerRenderer,
  onIssueAttachment: onIssueAttachmentTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
  getIssue: buildActionStateRegistry("retrieved"),
  updateIssue: buildActionStateRegistry("updated"),
  addIssueLabel: buildActionStateRegistry("labeled"),
  addIssueComment: buildActionStateRegistry("commented"),
  updateIssueComment: buildActionStateRegistry("updated"),
  createAttachment: buildActionStateRegistry("attached"),
  deleteAttachment: buildActionStateRegistry("removed"),
  removeIssueLabel: buildActionStateRegistry("unlabeled"),
  addReaction: buildActionStateRegistry("reacted"),
};
