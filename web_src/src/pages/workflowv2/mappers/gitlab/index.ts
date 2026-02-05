import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./component";
import { onIssueTriggerRenderer } from "./on_issue";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};
