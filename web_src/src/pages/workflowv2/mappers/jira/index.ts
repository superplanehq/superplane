import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { baseJiraMapper } from "./base";
import { onIssueCreatedTriggerRenderer } from "./on_issue_created";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: baseJiraMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssueCreated: onIssueCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
};
