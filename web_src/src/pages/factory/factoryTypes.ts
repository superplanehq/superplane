/**
 * Domain types for the Software Factory experience.
 *
 * Modelled directly on `docs/prd/software-factory.md`. Where the PRD leaves a
 * question open, the type keeps the option open rather than guessing — see the
 * comments on `trackedCost` and `attention`.
 */

/** PRD "Work Order Lifecycle" — the four confirmed states. */
export type WorkOrderState = "draft" | "ready" | "successful" | "unsuccessful";

/**
 * PRD groups Work Orders into four operational buckets. These are *derived*
 * from state plus attention/activity, not a fifth and sixth lifecycle state —
 * the PRD is explicit that only the four states above are confirmed.
 */
export type WorkOrderGroup = "needs-attention" | "running" | "recently-done" | "unsuccessful";

export interface PullRequestRef {
  provider: "github" | "bitbucket";
  repository: string;
  number: number;
  url: string;
}

/**
 * Why a Work Order is waiting on a person. The PRD lists this as an open
 * question ("What actions make a Work Order require attention"), so the reason
 * is free text rather than a closed enum.
 */
export interface WorkOrderAttention {
  reason: string;
  since: string;
}

export interface WorkOrder {
  id: string;
  title: string;
  description: string;
  state: WorkOrderState;
  /** Present when the Work Order is blocked on a human decision. */
  attention?: WorkOrderAttention;
  /** What an Automation is doing right now; present while work is moving. */
  activity?: string;
  /** Automation currently holding the Work Order. */
  currentAutomation?: string;
  /**
   * PRD: the initial interface emphasises one primary pull request, but a Work
   * Order must be able to retain several (e.g. frontend + backend).
   */
  pullRequests: PullRequestRef[];
  createdAt: string;
  updatedAt: string;
}

/** PRD "Work Order Event" — the durable, append-only chronology. */
export type WorkOrderEventKind =
  | "created"
  | "approved"
  | "pickup"
  | "progress"
  | "conversation"
  | "decision"
  | "approval-request"
  | "steering"
  | "handoff"
  | "pull-request"
  | "outcome"
  | "retry";

export interface WorkOrderEventActor {
  kind: "human" | "automation" | "system";
  name: string;
}

export interface WorkOrderEvent {
  id: string;
  kind: WorkOrderEventKind;
  at: string;
  actor: WorkOrderEventActor;
  /** Human-readable description of what happened — the PRD's minimum field. */
  summary: string;
  /** Longer body: a conversation message, a decision rationale, an error. */
  body?: string;
  /** Automation that produced the event, when one did. */
  automation?: string;
  pullRequest?: PullRequestRef;
  /** Set on `outcome` events. */
  outcome?: "successful" | "unsuccessful";
  /** Set on `approval-request` events still awaiting a response. */
  awaitingResponse?: boolean;
}

/** PRD "Automations Tab" — the row's required context. */
export interface Automation {
  id: string;
  name: string;
  description: string;
  trigger: string;
  status: "active" | "paused" | "error";
  /** What it is doing right now, if anything. */
  currentActivity?: string;
  /** Recent success rate, 0–1. */
  recentSuccess: number;
  lastRunAt: string;
  repositories: string[];
}

export interface FactorySummary {
  /** PRD leaves the throughput unit open; the label travels with the value. */
  throughput: { value: number; unit: string; trend: number[] };
  successRate: { value: number; trend: number[] };
  activeWorkOrders: number;
  /** Tracked cost = model tokens + execution compute only. */
  trackedCost: { tokensUsd: number; computeUsd: number };
}

export interface SoftwareFactory {
  id: string;
  name: string;
  description?: string;
  status: "healthy" | "degraded" | "paused";
  statusDetail?: string;
  automationCount: number;
}

/** PRD "Velocity Tab" — same indicators across three cohorts. */
export interface VelocityCohort {
  id: "team" | "human" | "factory";
  label: string;
  mergedPullRequests: number;
  /** Median cycle time in hours. */
  cycleTimeHours: number;
  successRate: number;
  /**
   * `null` means *not available*, which the PRD requires for human-authored
   * work — it must never read as "humans cost nothing".
   */
  trackedCostUsd: number | null;
}

export interface VelocityData {
  /** PRD: default period is the last 14 days. */
  periodDays: number;
  /** PRD: velocity is filtered by repository, never by Automation. */
  repositories: string[];
  selectedRepository: string;
  cohorts: VelocityCohort[];
  /** Throughput over time, one bucket per day of the period. */
  throughputSeries: { date: string; human: number; factory: number }[];
  costBreakdown: { tokensUsd: number; computeUsd: number };
}

export interface SoftwareFactoryPageData {
  factory: SoftwareFactory;
  summary: FactorySummary;
  workOrders: WorkOrder[];
  automations: Automation[];
  velocity: VelocityData;
}

export interface WorkOrderPageData {
  factory: Pick<SoftwareFactory, "id" | "name">;
  workOrder: WorkOrder;
  /** Oldest first — the page renders them in this order, top to bottom. */
  events: WorkOrderEvent[];
}

/** Derives the PRD's four operational groups from state + attention + activity. */
export function workOrderGroup(workOrder: WorkOrder): WorkOrderGroup {
  if (workOrder.attention) return "needs-attention";
  if (workOrder.state === "unsuccessful") return "unsuccessful";
  if (workOrder.state === "successful") return "recently-done";
  return "running";
}
