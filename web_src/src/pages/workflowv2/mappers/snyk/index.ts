import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { ignoreIssueMapper } from "./ignore_issue";
import { onNewIssueDetectedTriggerRenderer } from "./on_new_issue_detected";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  ignoreIssue: ignoreIssueMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onNewIssueDetected: onNewIssueDetectedTriggerRenderer,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};
