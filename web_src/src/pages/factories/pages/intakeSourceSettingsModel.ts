import type { FactoriesFactoryIntakeRun, FactoryIntakeSettings } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";

export type IntakeListenMode = "listen" | "schedule";
export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "runs" | "automation";

export function isIntakeSettingsTab(value: string | null | undefined): value is IntakeSettingsTab {
  return value === "general" || value === "runs" || value === "automation";
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
  listenMode: IntakeListenMode;
  confidencePct: number;
  labelFilterMode: IntakeLabelFilterMode;
  labels: string[];
  assignment: IntakeAssignmentFilter;
}

export const GITHUB_INTAKE_LABEL_OPTIONS = ["bug", "enhancement", "documentation", "good first issue"] as const;

export const DEFAULT_GITHUB_INTAKE_SETTINGS: IntakeSourceSettings = {
  name: "GitHub issues",
  listenMode: "listen",
  confidencePct: 65,
  labelFilterMode: "include",
  labels: [],
  assignment: "any",
};

export const INTAKE_SETTINGS_COPY = {
  title: "Intake GitHub issues",
  tabsLabel: "Intake settings",
  generalTab: "General",
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
  nameLabel: "Name",
  nameHelper: "Shown in the Intake list.",
  listenLabel: "When to analyze",
  listenOption: "Listen for new issues",
  listenHelper: "Analyze a GitHub issue when it is created.",
  scheduleOption: "Run on a schedule",
  scheduleHelper: "Scheduled intake is not available.",
  confidenceLabel: "Minimum confidence",
  confidenceHelper: "Move a ticket to Backlog when the score is this value or higher.",
  filtersLabel: "Filters",
  labelsLabel: "Labels",
  includeLabels: "Include these labels",
  excludeLabels: "Exclude these labels",
  labelsHelper: "Leave all labels off to match every issue.",
  assignmentLabel: "Assignment",
  assignmentAny: "Any assignment",
  assignmentAssigned: "Assigned",
  assignmentUnassigned: "Unassigned",
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
  const name = draft.name.trim() || DEFAULT_GITHUB_INTAKE_SETTINGS.name;
  const confidencePct = Math.min(100, Math.max(0, Math.round(draft.confidencePct)));
  return { ...draft, name, confidencePct };
}

export function intakeSettingsFromApi(name: string, settings: FactoryIntakeSettings | undefined): IntakeSourceSettings {
  return {
    name,
    listenMode: "listen",
    confidencePct: settings?.confidencePct ?? DEFAULT_GITHUB_INTAKE_SETTINGS.confidencePct,
    labelFilterMode: settings?.labelFilterMode === "LABEL_FILTER_MODE_EXCLUDE" ? "exclude" : "include",
    labels: settings?.labels ?? [],
    assignment: assignmentFromApi(settings?.assignment),
  };
}

export function intakeSettingsToApi(settings: IntakeSourceSettings): FactoryIntakeSettings {
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
  };
}

function assignmentFromApi(assignment: FactoryIntakeSettings["assignment"]): IntakeAssignmentFilter {
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
  plan: "plan",
  planning: "plan",
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

/** Runs the intake is still analyzing, shown as tickets under the source. */
export function analyzingTicketsFromApi(
  runs: FactoriesFactoryIntakeRun[],
  appId: string | undefined,
): Array<{ id: string; title: string; appId?: string; runId?: string }> {
  return runs.flatMap((run) => {
    const id = run.id?.trim();
    const title = run.title?.trim();
    if (!id || !title || run.placement !== "PLACEMENT_ANALYZING") {
      return [];
    }
    return [{ id, title, runId: id, ...(appId ? { appId } : {}) }];
  });
}

function minutesAgo(timestamp: string | undefined, now: Date): number {
  const value = timestamp ? Date.parse(timestamp) : Number.NaN;
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.floor((now.getTime() - value) / 60_000));
}
