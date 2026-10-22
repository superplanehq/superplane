import type { FactoriesFactoryIntakeSettings } from "@/api-client";

import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "agent" | "automation" | "runs";

export function isIntakeSettingsTab(value: string | null | undefined): value is IntakeSettingsTab {
  return value === "general" || value === "agent" || value === "automation" || value === "runs";
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
  defaultOwnerId: string;
  /** Create a task when a GitHub issue is created. */
  newIssues: boolean;
  /** Create a task when a closed GitHub issue is re-opened. */
  reopenedIssues: boolean;
  /** Also create a task when the include label, or "superplane" when none is set, is added to an open issue. */
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
  /** Sentry projects that still create a task. At least one is required. */
  sentryProjectIds: string[];
  /** Skip Productive.io key tasks (milestones). Productive task intakes only. */
  excludeKeyTasks: boolean;
  /** Productive.io task list ids that still create a task. Empty means every task list. */
  taskListIds: string[];
  /** Severities that still create a task. Empty means every severity. */
  dependabotSeverities: string[];
  /** Create a task when Datadog reports a Triggered Error Tracking alert. */
  datadogTriggeredAlerts: boolean;
  /** Create a task when Datadog reports a Re-Triggered Error Tracking alert. */
  datadogRetriggeredAlerts: boolean;
  /** Environments that still create a task. Empty means every environment. */
  datadogEnvironments: string[];
  /** Datadog service the trigger watches. Saved as the intake resource, not in settings. */
  datadogService: string;
  /** Linear team that creates tasks. Empty only for an older intake that has projects and no team. */
  linearTeamId: string;
  /** Linear projects that still create a task. Empty means every project of the team. */
  linearProjectIds: string[];
  /** Create a task for an issue without a project. Applies when projects are selected. */
  linearIncludeIssuesWithoutProject: boolean;
  /** Linear labels that still create a task. Empty means every label. */
  linearLabels: string[];
}

export const DEFAULT_GITHUB_INTAKE_SETTINGS: IntakeSourceSettings = {
  name: "GitHub issues",
  confidencePct: 65,
  labelFilterMode: "include",
  labels: [],
  filterByLabel: false,
  assignment: "any",
  defaultOwnerId: "",
  newIssues: true,
  reopenedIssues: true,
  superplaneLabelAdded: false,
  authorsWithAccess: true,
  jiraMoveOnComplete: true,
  jiraCompletionColumn: "",
  sentryNewIssues: true,
  sentryRegressedIssues: false,
  sentryAssignedIssues: false,
  sentryLevels: [],
  sentryProjectIds: [],
  excludeKeyTasks: true,
  taskListIds: [],
  dependabotSeverities: [],
  datadogTriggeredAlerts: true,
  datadogRetriggeredAlerts: false,
  datadogEnvironments: [],
  datadogService: "",
  linearTeamId: "",
  linearProjectIds: [],
  linearIncludeIssuesWithoutProject: true,
  linearLabels: [],
};

export const SENTRY_INTAKE_LEVELS = ["fatal", "error", "warning", "info", "debug"] as const;

export const DEPENDABOT_INTAKE_SEVERITIES = ["critical", "high", "medium", "low"] as const;

export const DEPENDABOT_INTAKE_SETTINGS_COPY = {
  severitySection: "Create a task for these severities:",
  critical: "Critical",
  high: "High",
  medium: "Medium",
  low: "Low",
} as const;

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
  name: "Productive tasks",
  excludeKeyTasks: true,
};

export const DEFAULT_DATADOG_INTAKE_SETTINGS: IntakeSourceSettings = {
  ...DEFAULT_GITHUB_INTAKE_SETTINGS,
  name: "Datadog errors",
  datadogTriggeredAlerts: true,
  datadogRetriggeredAlerts: false,
  datadogEnvironments: [],
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
  labelAddedHelper: "SuperPlane also creates a task when one of these labels is added to an open issue.",
  labelInput: "Search or add a label",
  labelPlaceholder: "Search labels",
  labelNamePlaceholder: "Label name",
  labelAdd: "Add label",
  labelsAllSelected: "All labels selected",
  labelsNoMatch: "No labels match the search.",
  labelsLoading: "Loading labels from the repository",
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
    sourceId === "dependabot-alerts" ||
    sourceId === "sentry-exceptions" ||
    sourceId === "jira-issues" ||
    sourceId === "productive-tasks" ||
    sourceId === "datadog" ||
    sourceId === "linear-issues"
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

/** Stable key for comparing intake settings from the server without reference churn. */
export function intakeSettingsContentKey(settings: IntakeSourceSettings): string {
  return JSON.stringify({
    ...settings,
    labels: [...settings.labels].sort(),
    taskListIds: [...settings.taskListIds].sort(),
    datadogEnvironments: [...settings.datadogEnvironments].sort(),
    linearProjectIds: [...settings.linearProjectIds].sort(),
    linearLabels: [...settings.linearLabels].sort(),
    sentryProjectIds: [...settings.sentryProjectIds].sort(),
    dependabotSeverities: [...settings.dependabotSeverities].sort(),
    sentryLevels: [...settings.sentryLevels].sort(),
  });
}

export function normalizeIntakeSourceSettings(
  draft: IntakeSourceSettings,
  sourceId?: LineIntakeSourceId,
): IntakeSourceSettings {
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  const sentryLevels = SENTRY_INTAKE_LEVELS.filter((level) => draft.sentryLevels.includes(level));
  const dependabotSeverities = normalizeDependabotSeverities(draft.dependabotSeverities);
  const hiddenSentryTriggers =
    sourceId === "sentry-exceptions" ? { sentryRegressedIssues: false, sentryAssignedIssues: false } : {};
  const taskListIds = normalizeTaskListIds(draft.taskListIds);
  const datadogEnvironments = normalizeDatadogEnvironments(draft.datadogEnvironments);
  const datadogService = draft.datadogService.trim();
  const linearTeamId = draft.linearTeamId.trim();
  const linearProjectIds = normalizeLinearValues(draft.linearProjectIds);
  const linearLabels = normalizeLinearValues(draft.linearLabels);
  const sentryProjectIds = normalizeLinearValues(draft.sentryProjectIds);
  const filterByLabel =
    sourceId === "github-issues" ? draft.filterByLabel || draft.labels.length > 0 : draft.filterByLabel;
  const labels = filterByLabel ? draft.labels : [];
  // GitHub intakes no longer offer an exclude mode. Force include on save so
  // an intake that still stores exclude keeps its labels and changes only
  // when the user saves the intake again.
  const labelFilterMode = sourceId === "github-issues" || !filterByLabel ? "include" : draft.labelFilterMode;
  const githubListening =
    sourceId === "github-issues"
      ? {
          newIssues: true,
          superplaneLabelAdded: filterByLabel && labels.length > 0,
        }
      : {};
  return {
    ...draft,
    ...hiddenSentryTriggers,
    ...githubListening,
    confidencePct,
    filterByLabel,
    labels,
    labelFilterMode,
    sentryLevels,
    dependabotSeverities,
    taskListIds,
    datadogEnvironments,
    datadogService,
    linearTeamId,
    linearProjectIds,
    linearLabels,
    sentryProjectIds,
  };
}

export function addIntakeEnvironment(environments: string[], environment: string): string[] {
  return normalizeDatadogEnvironments([...environments, environment]);
}

export function removeIntakeEnvironment(environments: string[], environment: string): string[] {
  const next = environment.trim().toLowerCase();
  return environments.filter((entry) => entry !== next);
}

export function normalizeDependabotSeverities(severities: string[]): string[] {
  const selected = DEPENDABOT_INTAKE_SEVERITIES.filter((severity) => severities.includes(severity));
  if (selected.length === 0 || selected.length === DEPENDABOT_INTAKE_SEVERITIES.length) {
    return [];
  }
  return [...selected];
}

export function dependabotSeveritySelected(severities: string[], severity: string): boolean {
  if (severities.length === 0) {
    return true;
  }
  return severities.includes(severity);
}

export function toggleDependabotSeverity(severities: string[], severity: string): string[] {
  const current = severities.length === 0 ? [...DEPENDABOT_INTAKE_SEVERITIES] : [...severities];
  const next = current.includes(severity) ? current.filter((entry) => entry !== severity) : [...current, severity];
  return normalizeDependabotSeverities(next);
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

function normalizeLinearValues(values: string[]): string[] {
  const normalized: string[] = [];
  for (const value of values) {
    const next = value.trim();
    if (next.length === 0 || normalized.includes(next)) {
      continue;
    }
    normalized.push(next);
  }
  return normalized;
}

/**
 * A Linear intake needs a team. An older intake that has projects and no team
 * can still save unchanged.
 */
export function linearIntakeScopeMissing(settings: IntakeSourceSettings): boolean {
  return settings.linearTeamId.trim() === "" && settings.linearProjectIds.length === 0;
}

export function linearProjectIdsFromResource(resourceId: string | undefined): string[] {
  return normalizeLinearValues((resourceId ?? "").split(","));
}

export function sentryProjectIdsFromResource(resourceId: string | undefined): string[] {
  return normalizeLinearValues((resourceId ?? "").split(","));
}

function normalizeDatadogEnvironments(environments: string[]): string[] {
  const normalized: string[] = [];
  for (const environment of environments) {
    const next = environment.trim().toLowerCase();
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
    defaultOwnerId: settings?.defaultOwnerId ?? "",
    confidencePct: settings?.confidencePct ?? DEFAULT_GITHUB_INTAKE_SETTINGS.confidencePct,
    labelFilterMode: settings?.labelFilterMode === "LABEL_FILTER_MODE_EXCLUDE" ? "exclude" : "include",
    labels,
    filterByLabel: labels.length > 0,
    assignment: assignmentFromApi(settings?.assignment),
    ...intakeTogglesFromApi(settings),
    jiraMoveOnComplete: settings?.jiraMoveOnComplete ?? DEFAULT_GITHUB_INTAKE_SETTINGS.jiraMoveOnComplete,
    jiraCompletionColumn: settings?.jiraCompletionColumn?.trim() ?? "",
    sentryLevels: SENTRY_INTAKE_LEVELS.filter((level) => (settings?.sentryLevels ?? []).includes(level)),
    sentryProjectIds: normalizeLinearValues(settings?.sentryProjectIds ?? []),
    dependabotSeverities: normalizeDependabotSeverities(settings?.dependabotSeverities ?? []),
    datadogTriggeredAlerts: settings?.datadogTriggeredAlerts ?? DEFAULT_DATADOG_INTAKE_SETTINGS.datadogTriggeredAlerts,
    datadogRetriggeredAlerts:
      settings?.datadogRetriggeredAlerts ?? DEFAULT_DATADOG_INTAKE_SETTINGS.datadogRetriggeredAlerts,
    datadogEnvironments: normalizeDatadogEnvironments(settings?.datadogEnvironments ?? []),
    datadogService: "",
    ...linearScopeFromApi(settings),
    ...productiveFiltersFromApi(settings),
  };
}

function linearScopeFromApi(
  settings: FactoriesFactoryIntakeSettings | undefined,
): Pick<
  IntakeSourceSettings,
  "linearTeamId" | "linearProjectIds" | "linearIncludeIssuesWithoutProject" | "linearLabels"
> {
  return {
    linearTeamId: settings?.linearTeamId?.trim() ?? "",
    linearProjectIds: normalizeLinearValues(settings?.linearProjectIds ?? []),
    linearIncludeIssuesWithoutProject:
      settings?.linearIncludeIssuesWithoutProject ?? DEFAULT_GITHUB_INTAKE_SETTINGS.linearIncludeIssuesWithoutProject,
    linearLabels: normalizeLinearValues(settings?.linearLabels ?? []),
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

export function githubIntakeLabelsForApi(settings: IntakeSourceSettings): string[] {
  if (settings.filterByLabel || settings.labels.length > 0) {
    return settings.labels;
  }
  return [];
}

export function intakeSettingsToApi(
  settings: IntakeSourceSettings,
  sourceId?: LineIntakeSourceId,
): FactoriesFactoryIntakeSettings {
  const labels =
    sourceId === "github-issues" ? githubIntakeLabelsForApi(settings) : settings.filterByLabel ? settings.labels : [];
  return {
    ...(sourceId === "github-issues" ? { defaultOwnerId: settings.defaultOwnerId } : {}),
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
    sentryProjectIds: normalizeLinearValues(settings.sentryProjectIds),
    dependabotSeverities: normalizeDependabotSeverities(settings.dependabotSeverities),
    excludeKeyTasks: settings.excludeKeyTasks,
    taskListIds: normalizeTaskListIds(settings.taskListIds),
    datadogTriggeredAlerts: settings.datadogTriggeredAlerts,
    datadogRetriggeredAlerts: settings.datadogRetriggeredAlerts,
    datadogEnvironments: normalizeDatadogEnvironments(settings.datadogEnvironments),
    linearTeamId: settings.linearTeamId.trim(),
    linearProjectIds: normalizeLinearValues(settings.linearProjectIds),
    linearIncludeIssuesWithoutProject: settings.linearIncludeIssuesWithoutProject,
    linearLabels: normalizeLinearValues(settings.linearLabels),
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
