import type { FactoriesFactoryIntakeSettings } from "@/api-client";

export type IntakeListenMode = "listen" | "schedule";
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
  listenMode: IntakeListenMode;
  confidencePct: number;
  labelFilterMode: IntakeLabelFilterMode;
  labels: string[];
  assignment: IntakeAssignmentFilter;
  authorsWithAccess: boolean;
}

export const GITHUB_INTAKE_LABEL_OPTIONS = ["bug", "enhancement", "documentation", "good first issue"] as const;

export const DEFAULT_GITHUB_INTAKE_SETTINGS: IntakeSourceSettings = {
  name: "GitHub issues",
  listenMode: "listen",
  confidencePct: 65,
  labelFilterMode: "include",
  labels: [],
  assignment: "any",
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
  nameLabel: "Name",
  nameHelper: "Shown in the Intake list.",
  listenLabel: "When to create",
  listenOption: "Listen for new issues",
  listenHelper: "Create a task when a GitHub issue is opened.",
  scheduleOption: "Run on a schedule",
  scheduleHelper: "Scheduled intake is not available.",
  filtersLabel: "Filters",
  labelsLabel: "Labels",
  includeLabels: "Include these labels",
  excludeLabels: "Exclude these labels",
  labelsHelper: "Leave all labels off to match every issue.",
  assignmentLabel: "Assignment",
  assignmentAny: "Any assignment",
  assignmentAssigned: "Assigned",
  assignmentUnassigned: "Unassigned",
  authorsLabel: "Authors",
  authorsWithAccess: "Only issues from people with repository access",
  authorsHelper: "Skip issues opened by outside contributors.",
  save: "Save",
  saving: "Saving",
  saveError: "SuperPlane could not save the intake settings. Try again.",
} as const;

export function toggleIntakeLabel(labels: string[], label: string): string[] {
  return labels.includes(label) ? labels.filter((entry) => entry !== label) : [...labels, label];
}

export function normalizeIntakeSourceSettings(draft: IntakeSourceSettings): IntakeSourceSettings {
  const name = draft.name.trim() || DEFAULT_GITHUB_INTAKE_SETTINGS.name;
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  return { ...draft, name, confidencePct };
}

export function intakeSettingsFromApi(
  name: string,
  settings: FactoriesFactoryIntakeSettings | undefined,
): IntakeSourceSettings {
  return {
    name,
    listenMode: "listen",
    confidencePct: settings?.confidencePct ?? DEFAULT_GITHUB_INTAKE_SETTINGS.confidencePct,
    labelFilterMode: settings?.labelFilterMode === "LABEL_FILTER_MODE_EXCLUDE" ? "exclude" : "include",
    labels: settings?.labels ?? [],
    assignment: assignmentFromApi(settings?.assignment),
    authorsWithAccess: settings?.authorsWithAccess ?? DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess,
  };
}

export function intakeSettingsToApi(settings: IntakeSourceSettings): FactoriesFactoryIntakeSettings {
  return {
    confidencePct: settings.confidencePct,
    labels: settings.labels,
    labelFilterMode: settings.labelFilterMode === "exclude" ? "LABEL_FILTER_MODE_EXCLUDE" : "LABEL_FILTER_MODE_INCLUDE",
    assignment:
      settings.assignment === "assigned"
        ? "ASSIGNMENT_ASSIGNED"
        : settings.assignment === "unassigned"
          ? "ASSIGNMENT_UNASSIGNED"
          : "ASSIGNMENT_ANY",
    authorsWithAccess: settings.authorsWithAccess,
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
