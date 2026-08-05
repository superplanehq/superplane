import type { WorkOrderTimelineCommentAutomation } from "../lib/workOrderTimelineEvents";

export interface CommentAuthorLabelInput {
  isAutomation: boolean;
  isSystem: boolean;
  automation: WorkOrderTimelineCommentAutomation | null | undefined;
  actorName: string | null | undefined;
}

//
// Resolve the display name shown next to "commented" in the timeline.
// Automation / system comments prefer the executing node's identity
// (node name + app) so reviewers can trace the comment back to the
// canvas node that wrote it, instead of showing a free-form author
// label. Human comments fall back to the resolved actor name.
//
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

function formatAutomationLabel(nodeName: string | undefined, appName: string | undefined): string | undefined {
  if (nodeName && appName) {
    return `${nodeName} · ${appName}`;
  }
  return nodeName || appName;
}

//
// Long-form label for an artifact kind, used inside the sentence
// "<actor> attached a <kind>" in the activity timeline body.
//
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
