import type { FactoriesFactoryIntakeSettings } from "@/api-client";

import { INTAKE_CONNECTION_COPY, intakeProviderDisplayName } from "./intakeConnectionModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeSettingsSectionId = "connection" | "triggers" | "labels" | "filters" | "factory" | "danger";

export interface IntakeSettingsSection {
  id: IntakeSettingsSectionId;
  label: string;
}

export function intakeSettingsSectionDomId(id: IntakeSettingsSectionId): string {
  return `intake-settings-${id}`;
}

export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "agent" | "automation";

export function isIntakeSettingsTab(value: string | null | undefined): value is IntakeSettingsTab {
  return value === "general" || value === "agent" || value === "automation";
}

export function intakeSettingsTabs(hasAgent: boolean): IntakeSettingsTab[] {
  return hasAgent ? ["general", "agent", "automation"] : ["general", "automation"];
}

export function resolveIntakeSettingsTab(
  tabs: readonly IntakeSettingsTab[],
  tab: IntakeSettingsTab,
): IntakeSettingsTab {
  if (tabs.includes(tab)) {
    return tab;
  }
  return "general";
}

export interface IntakeSourceSettings {
  name: string;
  confidencePct: number;
  labelFilterMode: IntakeLabelFilterMode;
  labels: string[];
  /** Show and apply the label chip list. Off means every issue matches. */
  filterByLabel: boolean;
  assignment: IntakeAssignmentFilter;
  /** Create a task when a GitHub issue is created. */
  newIssues: boolean;
  /** Create a task when a closed GitHub issue is re-opened. */
  reopenedIssues: boolean;
  /** Also create a task when somebody adds the "superplane" label to an open issue. */
  superplaneLabelAdded: boolean;
  authorsWithAccess: boolean;
  /** Move the originating Jira issue when SuperPlane completes the task. */
  jiraMoveOnComplete: boolean;
  /** Jira status name to move the issue to. Empty means the Done column. */
  jiraCompletionColumn: string;
  /** Create a task when a Sentry issue is created. */
  sentryNewIssues: boolean;
  /** Create a task when a Sentry issue becomes unresolved. */
  sentryRegressedIssues: boolean;
  /** Create a task when a Sentry issue is assigned. */
  sentryAssignedIssues: boolean;
  /** Issue levels that still create a task. Empty means every level. */
  sentryLevels: string[];
}

export const DEFAULT_GITHUB_INTAKE_SETTINGS: IntakeSourceSettings = {
  name: "GitHub issues",
  confidencePct: 65,
  labelFilterMode: "include",
  labels: [],
  filterByLabel: false,
  assignment: "any",
  newIssues: true,
  reopenedIssues: true,
  superplaneLabelAdded: true,
  authorsWithAccess: false,
  jiraMoveOnComplete: true,
  jiraCompletionColumn: "",
  sentryNewIssues: true,
  sentryRegressedIssues: true,
  sentryAssignedIssues: false,
  sentryLevels: [],
};

export const SENTRY_INTAKE_LEVELS = ["fatal", "error", "warning", "info", "debug"] as const;

export const DEFAULT_SENTRY_INTAKE_SETTINGS: IntakeSourceSettings = {
  ...DEFAULT_GITHUB_INTAKE_SETTINGS,
  name: "Sentry exceptions",
  sentryNewIssues: true,
  sentryRegressedIssues: true,
  sentryAssignedIssues: false,
  sentryLevels: [],
};

export const DEFAULT_JIRA_COMPLETION_SETTINGS = {
  jiraMoveOnComplete: true,
  jiraCompletionColumn: "",
} as const;

const INTAKE_SETTINGS_TITLE_BY_SOURCE: Record<LineIntakeSourceId, string> = {
  "github-issues": "GitHub intake",
  "jira-issues": "Jira intake",
  "sentry-exceptions": "Sentry intake",
  "pagerduty-incidents": "PagerDuty intake",
  "productive-tasks": "Productive intake",
};

export function intakeSettingsTitle(sourceId: LineIntakeSourceId): string {
  return INTAKE_SETTINGS_TITLE_BY_SOURCE[sourceId];
}

export const INTAKE_SETTINGS_COPY = {
  tabsLabel: "Intake settings",
  generalTab: "Settings",
  agentTab: "Agent",
  automationTab: "Canvas",
  editAutomation: "Edit automation",
  automationLoading: "The automation is loading.",
  automationEmpty: "This intake has no automation yet.",
  automationError: "SuperPlane could not load the automation.",
  retryAutomation: "Try again",
  intakeSection: "Create task when:",
  filtersLabel: "Filters",
  newIssues: "A new issue is opened",
  reopenedIssues: "A closed issue is re-opened",
  filterByLabel: "Issue has one of these labels",
  labelInput: "Issue label",
  labelPlaceholder: "Type a label name",
  labelNew: "Add label",
  labelAdd: "Add",
  labelCancel: "Cancel",
  labelsLoading: "Loading labels from the repository",
  labelsEmpty: "No labels found in the repository. Add a label name.",
  superplaneLabelAdded: 'The "superplane" label is added to the issue',
  authorsWithAccess: "Author is a repository collaborator",
  eventsThatCreateTasks: "Events that create tasks",
  eventsThatCreateTasksHelper: "The events you select here create tasks in the factory.",
  dangerZone: "Pause or delete",
  save: "Save",
  saving: "Saving",
  saveError: "SuperPlane could not save the intake settings. Try again.",
  pause: "Pause intake",
  resume: "Resume intake",
  pausing: "Pausing",
  resuming: "Resuming",
  delete: "Delete intake",
  deleteTitle: "Delete this intake?",
  deleteCancel: "Keep intake",
  deleteConfirm: "Delete intake",
  pauseError: "SuperPlane could not change the intake. Try again.",
  deleteError: "SuperPlane could not delete the intake. Try again.",
} as const;

export function intakeSettingsTabLabel(tab: IntakeSettingsTab): string {
  switch (tab) {
    case "general":
      return INTAKE_SETTINGS_COPY.generalTab;
    case "agent":
      return INTAKE_SETTINGS_COPY.agentTab;
    case "automation":
      return INTAKE_SETTINGS_COPY.automationTab;
  }
}

export function intakeSupportsPause(sourceId: LineIntakeSourceId): boolean {
  return (
    sourceId === "github-issues" ||
    sourceId === "sentry-exceptions" ||
    sourceId === "jira-issues" ||
    sourceId === "productive-tasks"
  );
}

function intakeStopsListeningCopy(sourceId: LineIntakeSourceId): string {
  return `SuperPlane stops listening for changes from ${intakeProviderDisplayName(sourceId)}.`;
}

export function intakePauseHelper(sourceId: LineIntakeSourceId): string {
  return `${intakeStopsListeningCopy(sourceId)} You can still import one item to Backlog by hand.`;
}

export function intakeDeleteHelper(sourceId: LineIntakeSourceId): string {
  return `${intakeStopsListeningCopy(sourceId)} Delete also removes this intake from Backlog. Tasks that this intake created stay in Backlog.`;
}

export function intakeDangerZoneHelper(sourceId: LineIntakeSourceId): string {
  return `Pause stops SuperPlane from listening for changes from ${intakeProviderDisplayName(sourceId)}. Delete removes this intake from Backlog.`;
}

export function intakeSettingsSections(sourceId: LineIntakeSourceId, hasConnection: boolean): IntakeSettingsSection[] {
  const sections: IntakeSettingsSection[] = [];

  if (hasConnection) {
    sections.push({ id: "connection", label: INTAKE_CONNECTION_COPY.section });
  }

  switch (sourceId) {
    case "github-issues":
      sections.push({ id: "triggers", label: "Triggers" });
      sections.push({ id: "filters", label: INTAKE_SETTINGS_COPY.filtersLabel });
      break;
    case "jira-issues":
      sections.push({ id: "triggers", label: INTAKE_SETTINGS_COPY.eventsThatCreateTasks });
      sections.push({ id: "labels", label: "Labels" });
      sections.push({ id: "factory", label: "When task completes" });
      break;
    case "sentry-exceptions":
      sections.push({ id: "triggers", label: INTAKE_SETTINGS_COPY.eventsThatCreateTasks });
      break;
    case "productive-tasks":
    case "pagerduty-incidents":
      break;
  }

  if (intakeSupportsPause(sourceId)) {
    sections.push({ id: "danger", label: INTAKE_SETTINGS_COPY.dangerZone });
  }

  return sections;
}

export function toggleIntakeLabel(labels: string[], label: string): string[] {
  return labels.includes(label) ? labels.filter((entry) => entry !== label) : [...labels, label];
}

export function addIntakeLabel(labels: string[], label: string): string[] {
  const next = label.trim();
  if (next.length === 0 || labels.includes(next)) {
    return labels;
  }
  return [...labels, next];
}
export function normalizeIntakeSourceSettings(draft: IntakeSourceSettings): IntakeSourceSettings {
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  const sentryLevels = SENTRY_INTAKE_LEVELS.filter((level) => draft.sentryLevels.includes(level));
  if (!draft.filterByLabel) {
    return { ...draft, confidencePct, labels: [], labelFilterMode: "include", sentryLevels };
  }
  return { ...draft, confidencePct, sentryLevels };
}

type IntakeToggles = Pick<
  IntakeSourceSettings,
  | "newIssues"
  | "reopenedIssues"
  | "superplaneLabelAdded"
  | "authorsWithAccess"
  | "sentryNewIssues"
  | "sentryRegressedIssues"
  | "sentryAssignedIssues"
>;

/** A response that omits a toggle predates it, so fall back to the default. */
function intakeTogglesFromApi(settings: FactoriesFactoryIntakeSettings | undefined): IntakeToggles {
  return {
    newIssues: settings?.newIssues ?? DEFAULT_GITHUB_INTAKE_SETTINGS.newIssues,
    reopenedIssues: settings?.reopenedIssues ?? DEFAULT_GITHUB_INTAKE_SETTINGS.reopenedIssues,
    superplaneLabelAdded: settings?.superplaneLabelAdded ?? DEFAULT_GITHUB_INTAKE_SETTINGS.superplaneLabelAdded,
    authorsWithAccess: settings?.authorsWithAccess ?? DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess,
    sentryNewIssues: settings?.sentryNewIssues ?? DEFAULT_SENTRY_INTAKE_SETTINGS.sentryNewIssues,
    sentryRegressedIssues: settings?.sentryRegressedIssues ?? DEFAULT_SENTRY_INTAKE_SETTINGS.sentryRegressedIssues,
    sentryAssignedIssues: settings?.sentryAssignedIssues ?? DEFAULT_SENTRY_INTAKE_SETTINGS.sentryAssignedIssues,
  };
}

export function intakeSettingsFromApi(
  name: string,
  settings: FactoriesFactoryIntakeSettings | undefined,
): IntakeSourceSettings {
  const labels = settings?.labels ?? [];
  return {
    name,
    confidencePct: settings?.confidencePct ?? DEFAULT_GITHUB_INTAKE_SETTINGS.confidencePct,
    labelFilterMode: settings?.labelFilterMode === "LABEL_FILTER_MODE_EXCLUDE" ? "exclude" : "include",
    labels,
    filterByLabel: labels.length > 0,
    assignment: assignmentFromApi(settings?.assignment),
    ...intakeTogglesFromApi(settings),
    jiraMoveOnComplete: settings?.jiraMoveOnComplete ?? DEFAULT_GITHUB_INTAKE_SETTINGS.jiraMoveOnComplete,
    jiraCompletionColumn: settings?.jiraCompletionColumn?.trim() ?? "",
    sentryLevels: SENTRY_INTAKE_LEVELS.filter((level) => (settings?.sentryLevels ?? []).includes(level)),
  };
}

export function intakeSettingsToApi(settings: IntakeSourceSettings): FactoriesFactoryIntakeSettings {
  const labels = settings.filterByLabel ? settings.labels : [];
  return {
    confidencePct: settings.confidencePct,
    labels,
    labelFilterMode: settings.labelFilterMode === "exclude" ? "LABEL_FILTER_MODE_EXCLUDE" : "LABEL_FILTER_MODE_INCLUDE",
    assignment:
      settings.assignment === "assigned"
        ? "ASSIGNMENT_ASSIGNED"
        : settings.assignment === "unassigned"
          ? "ASSIGNMENT_UNASSIGNED"
          : "ASSIGNMENT_ANY",
    authorsWithAccess: settings.authorsWithAccess,
    newIssues: settings.newIssues,
    reopenedIssues: settings.reopenedIssues,
    superplaneLabelAdded: settings.superplaneLabelAdded,
    jiraMoveOnComplete: settings.jiraMoveOnComplete,
    jiraCompletionColumn: settings.jiraMoveOnComplete ? settings.jiraCompletionColumn.trim() : "",
    sentryNewIssues: settings.sentryNewIssues,
    sentryRegressedIssues: settings.sentryRegressedIssues,
    sentryAssignedIssues: settings.sentryAssignedIssues,
    sentryLevels: SENTRY_INTAKE_LEVELS.filter((level) => settings.sentryLevels.includes(level)),
  };
}

export function jiraCompletionSettingsToApi(settings: {
  jiraMoveOnComplete: boolean;
  jiraCompletionColumn: string;
}): Pick<FactoriesFactoryIntakeSettings, "jiraMoveOnComplete" | "jiraCompletionColumn"> {
  return {
    jiraMoveOnComplete: settings.jiraMoveOnComplete,
    jiraCompletionColumn: settings.jiraMoveOnComplete ? settings.jiraCompletionColumn.trim() : "",
  };
}

function assignmentFromApi(assignment: FactoriesFactoryIntakeSettings["assignment"]): IntakeAssignmentFilter {
  if (assignment === "ASSIGNMENT_ASSIGNED") {
    return "assigned";
  }
  if (assignment === "ASSIGNMENT_UNASSIGNED") {
    return "unassigned";
  }
  return "any";
}
