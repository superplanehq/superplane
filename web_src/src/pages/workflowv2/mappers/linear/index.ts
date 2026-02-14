import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";
import { createIssueMapper } from "./create_issue";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
};
