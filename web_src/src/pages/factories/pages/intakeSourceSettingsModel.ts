import { formatTimeAgo } from "@/lib/date";

export type IntakeListenMode = "listen" | "schedule";
export type IntakeLabelFilterMode = "include" | "exclude";
export type IntakeAssignmentFilter = "any" | "assigned" | "unassigned";
export type IntakeSettingsTab = "general" | "runs" | "automation";
export type IntakeTicketPlacement = "backlog" | "rejected" | "progressed" | "below-threshold";
export type IntakeLineStage = "plan" | "implement" | "verify" | "done";

export interface IntakeAutomationRun {
  id: string;
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
  runsTab: "Runs",
  runsEmpty: "No runs yet.",
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
  stagePlan: "Plan",
  stageImplement: "Implement",
  stageVerify: "Verify",
  stageDone: "Done",
  nameLabel: "Name",
  nameHelper: "Shown in the Intake list.",
  listenLabel: "When to analyze",
  listenOption: "Listen for new issues",
  listenHelper: "Analyze a GitHub issue when it is created.",
  scheduleOption: "Run on a schedule",
  scheduleHelper: "Analyze open issues every few hours.",
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
} as const;

const STAGE_LABEL: Record<IntakeLineStage, string> = {
  plan: INTAKE_SETTINGS_COPY.stagePlan,
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
    stage: "plan",
    activity: "Drafting the 409 response plan.",
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
