import type { TriggerRenderer } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onIssue: onIssueTriggerRenderer,
};
