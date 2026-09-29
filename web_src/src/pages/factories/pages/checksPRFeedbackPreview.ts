import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

import { selectedSuggestedIntegrationNames, type ChecksToolsAccess } from "./checksPRFeedbackSetup";
import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";

export type ChecksPreviewRow = {
  name: string;
  selected: boolean;
};

export type ChecksPreviewToolKind = "granted" | "no-access" | "github-actions" | "none";

export type ChecksPreviewToolOutcome = {
  kind: ChecksPreviewToolKind;
  labels: string[];
};

const CHECK_PREVIEW_TOOL_LABELS: Record<string, string> = {
  circleci: "CircleCI",
  "github-actions": PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGitHub,
};

const CHECKS_PREVIEW_ROW_LIMIT = 4;

export function checksPreviewRows(
  catalogNames: string[],
  selected: string[],
  limit = CHECKS_PREVIEW_ROW_LIMIT,
): ChecksPreviewRow[] {
  const rows: ChecksPreviewRow[] = [];
  const seen = new Set<string>();

  for (const name of selected) {
    pushPreviewRow(rows, seen, name, true);
  }
  for (const name of catalogNames) {
    pushPreviewRow(rows, seen, name, false);
  }

  return rows.slice(0, limit);
}

function pushPreviewRow(rows: ChecksPreviewRow[], seen: Set<string>, name: string, selected: boolean) {
  const trimmed = name.trim();
  if (!trimmed) {
    return;
  }
  const key = trimmed.toLowerCase();
  if (seen.has(key)) {
    return;
  }
  seen.add(key);
  rows.push({ name: trimmed, selected });
}

export function checksPreviewAttemptsLabel(attempts: number): string {
  if (!Number.isInteger(attempts) || attempts < 1) {
    return "";
  }
  if (attempts === 1) {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksAttemptsOne;
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksAttemptsMany.replace("{count}", String(attempts));
}

export function checksPreviewToolLabel(name: string): string {
  const key = name.trim().toLowerCase();
  return CHECK_PREVIEW_TOOL_LABELS[key] ?? getIntegrationTypeDisplayName(undefined, name);
}

export function checksPreviewToolOutcome(input: {
  toolsAccess: ChecksToolsAccess;
  suggestedNames: string[];
  selectedIds: string[];
  connected: Array<{ metadata?: { id?: string; integrationName?: string } }>;
}): ChecksPreviewToolOutcome {
  if (input.toolsAccess === "suggested") {
    const grantedNames = selectedSuggestedIntegrationNames(input.suggestedNames, input.selectedIds, input.connected);
    if (grantedNames.length > 0) {
      return {
        kind: "granted",
        labels: grantedNames.map(checksPreviewToolLabel),
      };
    }
    return {
      kind: "no-access",
      labels: input.suggestedNames.map(checksPreviewToolLabel),
    };
  }
  if (input.toolsAccess === "github-actions") {
    return {
      kind: "github-actions",
      labels: [PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGitHub],
    };
  }
  return { kind: "none", labels: [] };
}

export function checksPreviewReadingLabel(labels: string[]): string {
  if (labels.length === 0) {
    return "";
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsReading.replace("{tool}", joinToolLabels(labels));
}

function joinToolLabels(labels: string[]): string {
  if (labels.length === 1) {
    return labels[0] ?? "";
  }
  if (labels.length === 2) {
    return `${labels[0]} and ${labels[1]}`;
  }
  return `${labels.slice(0, -1).join(", ")}, and ${labels[labels.length - 1]}`;
}

export function checksSetupPreviewCaption(input: {
  step: "checks" | "tools";
  catalogEmpty: boolean;
  selectedCount: number;
  toolOutcome?: ChecksPreviewToolOutcome;
}): string {
  if (input.step === "tools" && input.toolOutcome) {
    return checksToolsPreviewCaption(input.toolOutcome.kind);
  }
  if (input.catalogEmpty) {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksEmptyCaption;
  }
  if (input.selectedCount === 0) {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksNoneCaption;
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksSelectedCaption;
}

function checksToolsPreviewCaption(kind: ChecksPreviewToolKind): string {
  if (kind === "granted") {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGrantedCaption;
  }
  if (kind === "no-access") {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsNoAccessCaption;
  }
  if (kind === "github-actions") {
    return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsGitHubCaption;
  }
  return PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsNoneCaption;
}
