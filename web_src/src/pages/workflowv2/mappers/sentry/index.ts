import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueEventTriggerRenderer } from "./on_issue_event";
import { updateIssueMapper } from "./update_issue";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssueEvent: onIssueEventTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  updateIssue: buildActionStateRegistry("updated"),
};
