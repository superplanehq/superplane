import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueEventTriggerRenderer } from "./on_issue_event";
import { updateIssueMapper, UPDATE_ISSUE_STATE_REGISTRY } from "./update_issue";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssueEvent: onIssueEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  updateIssue: UPDATE_ISSUE_STATE_REGISTRY,
};
