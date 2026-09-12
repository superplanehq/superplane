import type { FactoriesFactoryIntakeSettings } from "@/api-client";

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
};

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
} as const;

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
  if (!draft.filterByLabel) {
    return { ...draft, confidencePct, labels: [], labelFilterMode: "include" };
  }
  return { ...draft, confidencePct };
}

type IntakeToggles = Pick<
  IntakeSourceSettings,
  "newIssues" | "reopenedIssues" | "superplaneLabelAdded" | "authorsWithAccess"
>;

/** A response that omits a toggle predates it, so fall back to the default. */
function intakeTogglesFromApi(settings: FactoriesFactoryIntakeSettings | undefined): IntakeToggles {
  return {
    newIssues: settings?.newIssues ?? DEFAULT_GITHUB_INTAKE_SETTINGS.newIssues,
    reopenedIssues: settings?.reopenedIssues ?? DEFAULT_GITHUB_INTAKE_SETTINGS.reopenedIssues,
    superplaneLabelAdded: settings?.superplaneLabelAdded ?? DEFAULT_GITHUB_INTAKE_SETTINGS.superplaneLabelAdded,
    authorsWithAccess: settings?.authorsWithAccess ?? DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess,
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
