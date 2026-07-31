import type { FactoriesWorkOrder } from "@/api-client";
import { hasActiveWorkOrderExecution, hasFailedWorkOrderExecution } from "./workOrderExecutions";

/** Derived lifecycle phase from assignees, closure state, and line executions. */
export type WorkOrderProgressPhase =
  | "unassigned"
  | "needs_attention"
  | "no_plan"
  | "planning"
  | "implementation_in_progress"
  | "verifications_running"
  | "verifications_failed"
  | "ready_for_review"
  | "completed"
  | "rejected";

export interface WorkOrderProgress {
  phase: WorkOrderProgressPhase;
  /** Short pill label shown next to the title (Plan, Verification, …). */
  stageLabel: string;
  /** Human-readable summary for section grouping. */
  summary: string;
}

export type WorkOrderSectionId = "needs_attention" | "unassigned" | "active" | "review" | "closed";

export interface WorkOrderSectionDefinition {
  id: WorkOrderSectionId;
  title: string;
  description: string;
  phases: WorkOrderProgressPhase[];
  tone?: "attention";
}

export const WORK_ORDER_SECTIONS: WorkOrderSectionDefinition[] = [
  {
    id: "needs_attention",
    title: "Needs attention",
    description: "Work paused for a decision, approval, or clarification.",
    phases: ["needs_attention"],
    tone: "attention",
  },
  {
    id: "unassigned",
    title: "Unassigned",
    description: "New intake waiting for someone to take ownership.",
    phases: ["unassigned"],
  },
  {
    id: "active",
    title: "In progress",
    description: "Planning, implementation, or verification underway.",
    phases: ["no_plan", "planning", "implementation_in_progress", "verifications_running", "verifications_failed"],
  },
  {
    id: "review",
    title: "Ready for review",
    description: "Implementation and verifications are done — waiting on a human.",
    phases: ["ready_for_review"],
  },
  {
    id: "closed",
    title: "Closed",
    description: "Completed or rejected work orders.",
    phases: ["completed", "rejected"],
  },
];

const PHASE_META: Record<WorkOrderProgressPhase, Pick<WorkOrderProgress, "stageLabel" | "summary">> = {
  unassigned: { stageLabel: "Intake", summary: "Waiting for an assignee" },
  needs_attention: { stageLabel: "Attention", summary: "Needs a human decision" },
  no_plan: { stageLabel: "Triage", summary: "Assigned — plan not started" },
  planning: { stageLabel: "Plan", summary: "Plan in progress" },
  implementation_in_progress: { stageLabel: "Implementation", summary: "Agent implementation running" },
  verifications_running: { stageLabel: "Verification", summary: "Verifications running" },
  verifications_failed: { stageLabel: "Verification", summary: "Verifications failed" },
  ready_for_review: { stageLabel: "Review", summary: "Ready for human review" },
  completed: { stageLabel: "Done", summary: "Completed" },
  rejected: { stageLabel: "Rejected", summary: "Rejected" },
};

export function getWorkOrderDisplayKey(order: FactoriesWorkOrder): string {
  if (order.id) {
    return order.id.slice(0, 8);
  }
  return "—";
}

export function deriveWorkOrderProgress(order: FactoriesWorkOrder): WorkOrderProgress {
  if (order.state === "STATE_CLOSED") {
    const phase: WorkOrderProgressPhase = order.result === "RESULT_REJECTED" ? "rejected" : "completed";
    return { phase, ...PHASE_META[phase] };
  }

  if (!order.assignees?.length) {
    return { phase: "unassigned", ...PHASE_META.unassigned };
  }

  const executions = order.executions ?? [];
  if (executions.length === 0) {
    return { phase: "no_plan", ...PHASE_META.no_plan };
  }

  if (hasActiveWorkOrderExecution(executions)) {
    return { phase: "implementation_in_progress", ...PHASE_META.implementation_in_progress };
  }

  if (hasFailedWorkOrderExecution(executions)) {
    return { phase: "verifications_failed", ...PHASE_META.verifications_failed };
  }

  return { phase: "ready_for_review", ...PHASE_META.ready_for_review };
}

export function isAssignedToUser(order: FactoriesWorkOrder, userId?: string): boolean {
  if (!userId) {
    return false;
  }
  return (order.assignees ?? []).some((assignee) => assignee.id === userId);
}

export function filterMyWorkOrders(orders: FactoriesWorkOrder[], userId?: string): FactoriesWorkOrder[] {
  if (!userId) {
    return [];
  }
  return orders.filter((order) => isAssignedToUser(order, userId) && order.state === "STATE_OPEN");
}

export function groupWorkOrdersBySection(
  orders: FactoriesWorkOrder[],
  sections: WorkOrderSectionDefinition[] = WORK_ORDER_SECTIONS,
): Array<{ section: WorkOrderSectionDefinition; orders: FactoriesWorkOrder[] }> {
  const progressByOrderId = new Map<string, WorkOrderProgressPhase>();
  for (const order of orders) {
    if (order.id) {
      progressByOrderId.set(order.id, deriveWorkOrderProgress(order).phase);
    }
  }

  return sections
    .map((section) => ({
      section,
      orders: orders.filter((order) => {
        if (!order.id) {
          return false;
        }
        const phase = progressByOrderId.get(order.id);
        return phase ? section.phases.includes(phase) : false;
      }),
    }))
    .filter((entry) => entry.orders.length > 0);
}

export function countOpenWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => order.state === "STATE_OPEN").length;
}

export function countNeedsAttention(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => deriveWorkOrderProgress(order).phase === "needs_attention").length;
}

export type WorkOrderDisplayStatus = "draft" | "ready" | "running" | "successful" | "unsuccessful";

const DISPLAY_STATUS_META: Record<WorkOrderDisplayStatus, { label: string; className: string; filterLabel: string }> = {
  draft: {
    label: "Draft",
    filterLabel: "Draft",
    className:
      "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200",
  },
  ready: {
    label: "Ready",
    filterLabel: "Ready",
    className: "border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200",
  },
  running: {
    label: "Running",
    filterLabel: "Running",
    className:
      "border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200",
  },
  successful: {
    label: "Successful",
    filterLabel: "Successful",
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200",
  },
  unsuccessful: {
    label: "Unsuccessful",
    filterLabel: "Unsuccessful",
    className: "border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200",
  },
};

export function getWorkOrderDisplayStatus(order: FactoriesWorkOrder): WorkOrderDisplayStatus {
  const progress = deriveWorkOrderProgress(order);
  switch (progress.phase) {
    case "unassigned":
      return "draft";
    case "no_plan":
    case "planning":
    case "ready_for_review":
      return "ready";
    case "implementation_in_progress":
    case "verifications_running":
    case "needs_attention":
      return "running";
    case "completed":
      return "successful";
    case "rejected":
    case "verifications_failed":
      return "unsuccessful";
    default:
      return "ready";
  }
}

export function getWorkOrderDisplayStatusMeta(status: WorkOrderDisplayStatus) {
  return DISPLAY_STATUS_META[status];
}

export type WorkOrderOwnerFilter = "all" | "mine";

export type WorkOrderStatusFilter = "all" | WorkOrderDisplayStatus;

export function filterWorkOrdersByOwner(
  orders: FactoriesWorkOrder[],
  ownerFilter: WorkOrderOwnerFilter,
  userId?: string,
): FactoriesWorkOrder[] {
  if (ownerFilter === "all") {
    return orders;
  }
  return filterMyWorkOrders(orders, userId);
}

export function filterWorkOrdersByStatus(
  orders: FactoriesWorkOrder[],
  statusFilter: WorkOrderStatusFilter,
): FactoriesWorkOrder[] {
  if (statusFilter === "all") {
    return orders;
  }
  return orders.filter((order) => getWorkOrderDisplayStatus(order) === statusFilter);
}

/** Open work on the factory queue — excludes closed orders. */
export const WORK_ORDER_TAB_SECTIONS = WORK_ORDER_SECTIONS.filter((section) => section.id !== "closed");

/** Assigned open work for the current user — no unassigned intake queue. */
export const MY_WORK_SECTIONS = WORK_ORDER_TAB_SECTIONS.filter((section) => section.id !== "unassigned");
