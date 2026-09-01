/**
 * Storybook-only mock data for the Workspace Overview redesign.
 *
 * The redesign targets the near-future product: the workspace both executes
 * work (attention queue, in flight, shipped) and proposes work (suggested
 * tasks, improvement proposals). Entities that do not exist in the
 * backend yet are modeled here as plain types so the mockup stays
 * self-contained.
 */

export type AttentionReason = "approval" | "question" | "failed" | "stalled";

export interface WorkOrderOwner {
  name: string;
  initials: string;
}

export interface AttentionItem {
  id: string;
  /** Workspace-scoped key, e.g. "RF-42". */
  workOrderKey: string;
  title: string;
  reason: AttentionReason;
  lineName: string;
  stepName: string;
  /** Precomputed wait duration label, e.g. "4h". */
  waitingFor: string;
  /** Task owner. Absent means anyone on the team can grab it. */
  owner?: WorkOrderOwner;
  /** True when the task is assigned to the viewer ("My" scope). */
  mine?: boolean;
}

export interface InFlightItem {
  id: string;
  workOrderKey: string;
  title: string;
  lineName: string;
  stepName: string;
  /** 1-based index of the current step. */
  stepIndex: number;
  stepCount: number;
  /** Precomputed elapsed label, e.g. "32m". */
  elapsed: string;
  /** True when the task is assigned to the viewer ("My" scope). */
  mine?: boolean;
}

export type ShippedOutcome = "merged" | "in-review" | "unsuccessful";

export interface ShippedItem {
  id: string;
  workOrderKey: string;
  title: string;
  outcome: ShippedOutcome;
  /** Context line, e.g. "PR #482 · superplane/superplane". */
  detail: string;
  /** Precomputed relative time label, e.g. "2h ago". */
  when: string;
  /** True when the task is assigned to the viewer ("My" scope). */
  mine?: boolean;
}

export type MetricTone = "positive" | "negative" | "neutral";

export interface HealthMetric {
  id: string;
  label: string;
  value: string;
  /** Change versus the previous period, e.g. "+12%". */
  delta: string;
  tone: MetricTone;
}

export interface WorkOrderCandidate {
  id: string;
  title: string;
  repository: string;
  confidencePct: number;
}

export type ImprovementCategory = "tests" | "context" | "line";

export interface ImprovementProposal {
  id: string;
  category: ImprovementCategory;
  /** Short imperative title, e.g. "Add a CI check step to the Backend line". */
  title: string;
  /** One sentence on why the proposal matters. */
  description: string;
  /** Estimated effect on the readiness score, e.g. "+7 readiness". */
  impactLabel: string;
  /** Action the proposal offers, e.g. "Update line" or "Create task". */
  actionLabel: string;
}

export interface ReadinessCategoryScore {
  id: ImprovementCategory;
  label: string;
  score: number;
}

export interface WorkspaceReadiness {
  overall: number;
  categories: ReadinessCategoryScore[];
}

export interface BriefingCounts {
  attention: number;
  inFlight: number;
}

export interface SuggestionsState {
  /** When true, the repository scan runs and no candidates exist yet. */
  scanning: boolean;
  scanTarget?: string;
  candidates: WorkOrderCandidate[];
}

export interface OverviewRedesignData {
  /** When present, the header renders the briefing line instead of a static subtitle. */
  briefing?: BriefingCounts;
  attention: AttentionItem[];
  inFlight: InFlightItem[];
  shipped: ShippedItem[];
  health?: HealthMetric[];
  suggestions: SuggestionsState;
  improvements: ImprovementProposal[];
  /** AI readiness analysis. Absent when no repository is connected. */
  readiness?: WorkspaceReadiness;
}

const POPULATED_HEALTH: HealthMetric[] = [
  { id: "merged", label: "Merged PRs", value: "14", delta: "+12%", tone: "positive" },
  { id: "waste", label: "Waste", value: "9%", delta: "-3pts", tone: "positive" },
  { id: "cost", label: "Cost", value: "$128", delta: "$3.80 / PR", tone: "neutral" },
  { id: "share", label: "SuperPlane share", value: "26%", delta: "+4pts", tone: "positive" },
];

export const POPULATED_OVERVIEW: OverviewRedesignData = {
  briefing: { attention: 3, inFlight: 4 },
  attention: [
    {
      id: "att-1",
      workOrderKey: "RF-61",
      title: "Migrate refund webhooks to the new event schema",
      reason: "approval",
      lineName: "Backend",
      stepName: "Plan review",
      waitingFor: "4h",
      owner: { name: "Leonardo DiCaprio", initials: "LD" },
      mine: true,
    },
    {
      id: "att-2",
      workOrderKey: "RF-58",
      title: "Add retry limits to the payment poller",
      reason: "question",
      lineName: "Backend",
      stepName: "Build",
      waitingFor: "1h",
    },
    {
      id: "att-3",
      workOrderKey: "RF-54",
      title: "Fix flaky checkout E2E test on slow networks",
      reason: "failed",
      lineName: "Frontend",
      stepName: "CI check",
      waitingFor: "26m",
      owner: { name: "Luka Novak", initials: "LN" },
    },
  ],
  inFlight: [
    {
      id: "run-1",
      workOrderKey: "RF-63",
      title: "Add audit log entries for refund overrides",
      lineName: "Backend",
      stepName: "Build",
      stepIndex: 2,
      stepCount: 5,
      elapsed: "18m",
      mine: true,
    },
    {
      id: "run-2",
      workOrderKey: "RF-62",
      title: "Bump Go toolchain and fix deprecations",
      lineName: "Maintenance",
      stepName: "CI check",
      stepIndex: 3,
      stepCount: 4,
      elapsed: "42m",
    },
    {
      id: "run-3",
      workOrderKey: "RF-60",
      title: "Support CSV export on the disputes table",
      lineName: "Frontend",
      stepName: "Review",
      stepIndex: 4,
      stepCount: 5,
      elapsed: "2h",
      mine: true,
    },
    {
      id: "run-4",
      workOrderKey: "RF-59",
      title: "Cache exchange rates for refund conversions",
      lineName: "Backend",
      stepName: "Plan",
      stepIndex: 1,
      stepCount: 5,
      elapsed: "6m",
    },
  ],
  shipped: [
    {
      id: "ship-1",
      workOrderKey: "RF-57",
      title: "Return clear errors for expired refund tokens",
      outcome: "merged",
      detail: "PR #482 · superplane/superplane",
      when: "2h ago",
      mine: true,
    },
    {
      id: "ship-2",
      workOrderKey: "RF-56",
      title: "Add pagination to the refunds list endpoint",
      outcome: "in-review",
      detail: "PR #479 · superplane/superplane",
      when: "5h ago",
    },
    {
      id: "ship-3",
      workOrderKey: "RF-53",
      title: "Dedupe customer notification emails",
      outcome: "merged",
      detail: "PR #474 · superplane/notifications",
      when: "1d ago",
    },
    {
      id: "ship-4",
      workOrderKey: "RF-51",
      title: "Rewrite the ledger reconciliation job",
      outcome: "unsuccessful",
      detail: "Closed after 3 attempts",
      when: "2d ago",
      mine: true,
    },
    {
      id: "ship-5",
      workOrderKey: "RF-49",
      title: "Log webhook delivery latency per provider",
      outcome: "merged",
      detail: "PR #468 · superplane/superplane",
      when: "3d ago",
    },
  ],
  health: POPULATED_HEALTH,
  suggestions: {
    scanning: false,
    candidates: [
      {
        id: "cand-1",
        title: "Remove the deprecated v1 refund endpoint and its callers",
        repository: "superplane/superplane",
        confidencePct: 92,
      },
      {
        id: "cand-2",
        title: "Add input validation to the currency conversion helper",
        repository: "superplane/superplane",
        confidencePct: 81,
      },
      {
        id: "cand-3",
        title: "Close the race condition in webhook retries",
        repository: "superplane/notifications",
        confidencePct: 64,
      },
    ],
  },
  improvements: [
    {
      id: "imp-1",
      category: "line",
      title: "Add a CI check step to the Backend line",
      description: "Agents can repair failing builds when the line runs CI.",
      impactLabel: "+7 readiness",
      actionLabel: "Update line",
    },
    {
      id: "imp-2",
      category: "tests",
      title: "Increase test coverage in pkg/workers",
      description: "Coverage is 41%. Low coverage limits how much the workspace can automate.",
      impactLabel: "+9 readiness",
      actionLabel: "Create automation",
    },
    {
      id: "imp-3",
      category: "context",
      title: "Describe the release process in AGENTS.md",
      description: "Agents lack release context and ask more questions than needed.",
      impactLabel: "+5 readiness",
      actionLabel: "Create task",
    },
  ],
  readiness: {
    overall: 62,
    categories: [
      { id: "tests", label: "Tests", score: 41 },
      { id: "context", label: "Agent context", score: 74 },
      { id: "line", label: "Line setup", score: 58 },
    ],
  },
};

export const QUIET_OVERVIEW: OverviewRedesignData = {
  briefing: { attention: 0, inFlight: 2 },
  attention: [],
  inFlight: POPULATED_OVERVIEW.inFlight.slice(0, 2),
  shipped: POPULATED_OVERVIEW.shipped,
  health: POPULATED_HEALTH,
  suggestions: POPULATED_OVERVIEW.suggestions,
  improvements: POPULATED_OVERVIEW.improvements,
  readiness: POPULATED_OVERVIEW.readiness,
};

export const FRESH_OVERVIEW: OverviewRedesignData = {
  attention: [],
  inFlight: [],
  shipped: [],
  suggestions: {
    scanning: true,
    scanTarget: "superplane/superplane",
    candidates: [],
  },
  improvements: [],
};
