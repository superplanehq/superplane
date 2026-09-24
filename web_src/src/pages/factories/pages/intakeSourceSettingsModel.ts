import type { FactoriesFactoryIntakeSettings } from "@/api-client";

import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "agent" | "automation";

export function isIntakeSettingsTab(value: string | null | undefined): value is IntakeSettingsTab {
  return value === "general" || value === "agent" || value === "automation";
}

export function intakeSettingsTabs(hasAgent: boolean): IntakeSettingsTab[] {
  return hasAgent ? ["general", "agent", "automation"] : ["general", "automation"];
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
  /** Skip Productive.io key tasks (milestones). Productive task intakes only. */
  excludeKeyTasks: boolean;
  /** Productive.io task list ids that still create a task. Empty means every task list. */
  taskListIds: string[];
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
  sentryRegressedIssues: false,
  sentryAssignedIssues: false,
  sentryLevels: [],
  excludeKeyTasks: true,
  taskListIds: [],
};

export const SENTRY_INTAKE_LEVELS = ["fatal", "error", "warning", "info", "debug"] as const;

export const DEFAULT_SENTRY_INTAKE_SETTINGS: IntakeSourceSettings = {
  ...DEFAULT_GITHUB_INTAKE_SETTINGS,
  name: "Sentry exceptions",
  sentryNewIssues: true,
  sentryRegressedIssues: false,
  sentryAssignedIssues: false,
  sentryLevels: [],
};

export const DEFAULT_PRODUCTIVE_INTAKE_SETTINGS: IntakeSourceSettings = {
  ...DEFAULT_GITHUB_INTAKE_SETTINGS,
  name: "Productive.io tasks",
  excludeKeyTasks: true,
};

export const DEFAULT_JIRA_COMPLETION_SETTINGS = {
  jiraMoveOnComplete: true,
  jiraCompletionColumn: "",
} as const;

export const INTAKE_SETTINGS_COPY = {
  title: "Intake GitHub issues",
  tabsLabel: "Intake settings",
  generalTab: "General",
  agentTab: "Agent",
  automationTab: "Automation",
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
  save: "Save",
  saving: "Saving",
  saveError: "SuperPlane could not save the intake settings. Try again.",
  delete: "Delete intake",
  deleteTitle: "Delete this intake?",
  deleteDescription:
    "SuperPlane stops new items and removes this intake from Backlog. Tasks that it created stay in Backlog.",
  deleteCancel: "Keep intake",
  deleteConfirm: "Delete intake",
  deleteError: "SuperPlane could not delete the intake. Try again.",
} as const;

export function intakeSupportsDelete(sourceId: LineIntakeSourceId): boolean {
  return (
    sourceId === "github-issues" ||
    sourceId === "sentry-exceptions" ||
    sourceId === "jira-issues" ||
    sourceId === "productive-tasks"
  );
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
export function normalizeIntakeSourceSettings(
  draft: IntakeSourceSettings,
  sourceId?: LineIntakeSourceId,
): IntakeSourceSettings {
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  const sentryLevels = SENTRY_INTAKE_LEVELS.filter((level) => draft.sentryLevels.includes(level));
  const hiddenSentryTriggers =
    sourceId === "sentry-exceptions" ? { sentryRegressedIssues: false, sentryAssignedIssues: false } : {};
  const taskListIds = normalizeTaskListIds(draft.taskListIds);
  if (!draft.filterByLabel) {
    return {
      ...draft,
      ...hiddenSentryTriggers,
      confidencePct,
      labels: [],
      labelFilterMode: "include",
      sentryLevels,
      taskListIds,
    };
  }
  return { ...draft, ...hiddenSentryTriggers, confidencePct, sentryLevels, taskListIds };
}

function normalizeTaskListIds(ids: string[]): string[] {
  const normalized: string[] = [];
  for (const id of ids) {
    const next = id.trim();
    if (next.length === 0 || normalized.includes(next)) {
      continue;
    }
    normalized.push(next);
  }
  return normalized;
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
    ...productiveFiltersFromApi(settings),
  };
}

function productiveFiltersFromApi(
  settings: FactoriesFactoryIntakeSettings | undefined,
): Pick<IntakeSourceSettings, "excludeKeyTasks" | "taskListIds"> {
  return {
    excludeKeyTasks: settings?.excludeKeyTasks ?? DEFAULT_PRODUCTIVE_INTAKE_SETTINGS.excludeKeyTasks,
    taskListIds: normalizeTaskListIds(settings?.taskListIds ?? []),
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
    sentryRegressedIssues: false,
    sentryAssignedIssues: false,
    sentryLevels: SENTRY_INTAKE_LEVELS.filter((level) => settings.sentryLevels.includes(level)),
    excludeKeyTasks: settings.excludeKeyTasks,
    taskListIds: normalizeTaskListIds(settings.taskListIds),
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
