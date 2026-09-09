import type { FactoriesFactoryIntakeRun, FactoriesFactoryIntakeSettings } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";

export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "agent" | "runs" | "automation";

export function isIntakeSettingsTab(value: string | null | undefined): value is IntakeSettingsTab {
  return value === "general" || value === "agent" || value === "runs" || value === "automation";
}

export function intakeSettingsTabs(hasAgent: boolean): IntakeSettingsTab[] {
  return hasAgent ? ["general", "agent", "runs", "automation"] : ["general", "runs", "automation"];
}
export type IntakeTicketPlacement = "backlog" | "rejected" | "progressed" | "below-threshold";
export type IntakeLineStage = "implement" | "verify" | "done";

export interface IntakeAutomationRun {
  id: string;
  appId?: string;
  runId?: string;
  title: string;
  confidencePct: number;
  ranMinutesAgo: number;
  analyzedMinutesAgo: number;
  placement: IntakeTicketPlacement;
  stage?: IntakeLineStage;
  activity?: string;
}

export interface IntakeSourceSettings {
  name: string;
  confidencePct: number;
  labelFilterMode: IntakeLabelFilterMode;
  labels: string[];
  /** Show and apply the label chip list. Off means every issue matches. */
  filterByLabel: boolean;
  assignment: IntakeAssignmentFilter;
  /** Create a task when a GitHub issue is created or re-opened. */
  newIssues: boolean;
  /** Also create a task when an open issue is assigned to the agent. */
  assignedToAgent: boolean;
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
  assignedToAgent: false,
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
  runsTab: "Runs",
  runsEmpty: "No runs yet.",
  runsLoading: "Runs are loading.",
  runsError: "SuperPlane could not load the runs.",
  retryRuns: "Try again",
  runWhen: "Run",
  analysisWhen: "Analysis",
  scoreWhen: "Score",
  viewRun: "View run",
  viewRunFor: (title: string) => `View run for ${title}`,
  inBacklog: "In Backlog",
  backlogActivity: "Waiting for review.",
  rejected: "Rejected",
  rejectedActivity: "A person rejected this ticket.",
  belowThreshold: "Not moved to Backlog",
  belowThresholdActivity: "Score is below the minimum confidence.",
  stageImplement: "Implement",
  stageVerify: "Verify",
  stageDone: "Done",
  intakeSection: "Create tasks from",
  filtersLabel: "Filters",
  newIssues: "New and re-opened issues",
  filterByLabel: "Only issues with any of these labels",
  labelInput: "Issue label",
  labelPlaceholder: "Type a label name",
  labelNew: "Add label",
  labelAdd: "Add",
  labelCancel: "Cancel",
  labelsLoading: "Loading labels from the repository",
  labelsEmpty: "No labels found in the repository. Add a label name.",
  assignedToAgent: "Issues assigned to @superplaneagent",
  authorsWithAccess: "Only issues from people with repository access",
  save: "Save",
  saving: "Saving",
  saveError: "SuperPlane could not save the intake settings. Try again.",
} as const;

const STAGE_LABEL: Record<IntakeLineStage, string> = {
  implement: INTAKE_SETTINGS_COPY.stageImplement,
  verify: INTAKE_SETTINGS_COPY.stageVerify,
  done: INTAKE_SETTINGS_COPY.stageDone,
};

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

export const GITHUB_INTAKE_RUNS: IntakeAutomationRun[] = [
  {
    id: "gh-issue-1",
    title: "Handle duplicate refunds on retry",
    confidencePct: 94,
    ranMinutesAgo: 180,
    analyzedMinutesAgo: 170,
    placement: "progressed",
    stage: "implement",
    activity: "Writing the retry handler.",
  },
  {
    id: "gh-issue-2",
    title: "Return 409 when the invoice is already paid",
    confidencePct: 88,
    ranMinutesAgo: 120,
    analyzedMinutesAgo: 110,
    placement: "progressed",
    stage: "verify",
    activity: "Checking the 409 response.",
  },
  {
    id: "gh-issue-3",
    title: "Show a clearer empty state on the billing page",
    confidencePct: 81,
    ranMinutesAgo: 90,
    analyzedMinutesAgo: 80,
    placement: "backlog",
  },
  {
    id: "gh-issue-4",
    title: "Upgrade the Node 20 base image",
    confidencePct: 76,
    ranMinutesAgo: 45,
    analyzedMinutesAgo: 40,
    placement: "rejected",
  },
  {
    id: "gh-issue-5",
    title: "Add a flake retry to the checkout e2e suite",
    confidencePct: 68,
    ranMinutesAgo: 20,
    analyzedMinutesAgo: 15,
    placement: "backlog",
  },
  {
    id: "gh-issue-6",
    title: "Document the refund webhook contract",
    confidencePct: 52,
    ranMinutesAgo: 8,
    analyzedMinutesAgo: 5,
    placement: "below-threshold",
  },
];

export function intakeRelativeTime(minutesAgo: number): string {
  return formatTimeAgo(new Date(Date.now() - minutesAgo * 60_000));
}

export function intakeStageLabel(stage: IntakeLineStage): string {
  return STAGE_LABEL[stage];
}

export function intakePlacementLabel(run: IntakeAutomationRun): string {
  if (run.placement === "progressed" && run.stage) {
    return intakeStageLabel(run.stage);
  }
  if (run.placement === "rejected") {
    return INTAKE_SETTINGS_COPY.rejected;
  }
  if (run.placement === "below-threshold") {
    return INTAKE_SETTINGS_COPY.belowThreshold;
  }
  return INTAKE_SETTINGS_COPY.inBacklog;
}

export function intakePlacementActivity(run: IntakeAutomationRun): string {
  if (run.placement === "progressed") {
    return run.activity ?? "";
  }
  if (run.placement === "rejected") {
    return INTAKE_SETTINGS_COPY.rejectedActivity;
  }
  if (run.placement === "below-threshold") {
    return INTAKE_SETTINGS_COPY.belowThresholdActivity;
  }
  return INTAKE_SETTINGS_COPY.backlogActivity;
}

export function normalizeIntakeSourceSettings(draft: IntakeSourceSettings): IntakeSourceSettings {
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  if (!draft.filterByLabel) {
    return { ...draft, confidencePct, labels: [], labelFilterMode: "include" };
  }
  return { ...draft, confidencePct };
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
    newIssues: settings?.newIssues ?? DEFAULT_GITHUB_INTAKE_SETTINGS.newIssues,
    assignedToAgent: settings?.assignedToAgent ?? DEFAULT_GITHUB_INTAKE_SETTINGS.assignedToAgent,
    authorsWithAccess: settings?.authorsWithAccess ?? DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess,
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
    assignedToAgent: settings.assignedToAgent,
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

const PLACEMENT_BY_API: Record<string, IntakeTicketPlacement> = {
  PLACEMENT_BACKLOG: "backlog",
  PLACEMENT_REJECTED: "rejected",
  PLACEMENT_PROGRESSED: "progressed",
  PLACEMENT_BELOW_THRESHOLD: "below-threshold",
};

const STAGE_BY_NAME: Record<string, IntakeLineStage> = {
  plan: "implement",
  planning: "implement",
  implement: "implement",
  implementation: "implement",
  verify: "verify",
  verification: "verify",
  done: "done",
};

/**
 * The server decides placement, confidence, and stage. This only turns the
 * response into the shape the list renders, and drops runs that are still
 * being analyzed: those belong in the Analyzing list.
 */
export function intakeRunsFromApi(
  runs: FactoriesFactoryIntakeRun[],
  appId: string | undefined,
  now = new Date(),
): IntakeAutomationRun[] {
  return runs.flatMap((run) => {
    const id = run.id?.trim();
    const title = run.title?.trim();
    const placement = run.placement ? PLACEMENT_BY_API[run.placement] : undefined;
    if (!id || !title || !placement) {
      return [];
    }

    const stage = run.stage ? STAGE_BY_NAME[run.stage.trim().toLowerCase()] : undefined;
    return [
      {
        id,
        runId: id,
        ...(appId ? { appId } : {}),
        title,
        confidencePct: run.confidencePct ?? 0,
        ranMinutesAgo: minutesAgo(run.createdAt, now),
        analyzedMinutesAgo: minutesAgo(run.analyzedAt ?? run.createdAt, now),
        placement,
        ...(stage ? { stage } : {}),
      },
    ];
  });
}

function minutesAgo(timestamp: string | undefined, now: Date): number {
  const value = timestamp ? Date.parse(timestamp) : Number.NaN;
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.floor((now.getTime() - value) / 60_000));
}
