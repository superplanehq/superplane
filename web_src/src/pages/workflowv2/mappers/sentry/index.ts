import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";
import { updateIssueMapper } from "./update_issue";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  updateIssue: updateIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  updateIssue: buildActionStateRegistry("updated"),
};
