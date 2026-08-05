import type { WorkOrderTimelineCommentAutomation } from "../lib/workOrderTimelineEvents";

export interface CommentAuthorLabelInput {
  isAutomation: boolean;
  isSystem: boolean;
  automation: WorkOrderTimelineCommentAutomation | null | undefined;
  actorName: string | null | undefined;
}

// Automation/system comments show the executing node's identity;
// human comments fall back to the resolved actor name.
export function resolveCommentAuthorLabel({
  isAutomation,
  isSystem,
  automation,
  actorName,
}: CommentAuthorLabelInput): string {
  const nodeName = automation?.nodeName?.trim();
  const appName = automation?.appName?.trim();
  const automationLabel = formatAutomationLabel(nodeName, appName);

  if (isAutomation) {
    return automationLabel || "Automation";
  }
  if (isSystem) {
    return automationLabel || "System";
  }
  return actorName || "Someone";
}

export function formatAutomationLabel(nodeName: string | undefined, appName: string | undefined): string | undefined {
  if (nodeName && appName) {
    return `${nodeName} · ${appName}`;
  }
  return nodeName || appName;
}

// Long-form label used in "<actor> attached a <kind>".
const ARTIFACT_KIND_LONG_LABEL: Record<string, string> = {
  pr: "pull request",
  markdown: "note",
};

export function formatArtifactKindLong(type: string | null | undefined): string {
  if (!type) {
    return "artifact";
  }
  return ARTIFACT_KIND_LONG_LABEL[type] ?? "artifact";
}
