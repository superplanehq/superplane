import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { onIssueCreatedTriggerRenderer } from "./on_issue_created";
import { createIssueMapper } from "./create_issue";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssueCreated: onIssueCreatedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
};
