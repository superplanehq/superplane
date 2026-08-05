import type { FactoriesWorkOrder } from "@/api-client";
import { hasActiveWorkOrderExecution, latestFinishedWorkOrderExecution } from "./workOrderExecutions";

export type WorkOrderDisplayStatus =
  | "draft"
  | "ready"
  | "open"
  | "running"
  | "failed"
  | "completed"
  | "rejected"
  | "closedFailed";

const DISPLAY_STATUS_META: Record<
  WorkOrderDisplayStatus,
  { label: string; filterLabel: string; summary: string; className: string }
> = {
  draft: {
    label: "Draft",
    filterLabel: "Draft",
    summary: "Being scoped — not ready to run",
    className: "border-gray-200 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300",
  },
  ready: {
    label: "Ready",
    filterLabel: "Ready",
    summary: "Ready to dispatch",
    className:
      "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200",
  },
  open: {
    label: "Open",
    filterLabel: "Open",
    summary: "Ready to dispatch or between runs",
    className: "border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200",
  },
  running: {
    label: "Running",
    filterLabel: "Running",
    summary: "Line execution in progress",
    className:
      "border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200",
  },
  failed: {
    label: "Failed",
    filterLabel: "Failed",
    summary: "A line step failed",
    className: "border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200",
  },
  completed: {
    label: "Completed",
    filterLabel: "Completed",
    summary: "Work order completed",
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200",
  },
  rejected: {
    label: "Rejected",
    filterLabel: "Rejected",
    summary: "Work order rejected",
    className: "border-gray-200 bg-gray-50 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300",
  },
  closedFailed: {
    label: "Failed",
    filterLabel: "Failed",
    summary: "Work order closed as failed",
    className: "border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200",
  },
};

export type WorkOrderSectionId = "failed" | "running" | "open" | "draft" | "ready" | "closed";

export interface WorkOrderSectionDefinition {
  id: WorkOrderSectionId;
  title: string;
  description: string;
  statuses: WorkOrderDisplayStatus[];
  tone?: "attention";
}

export const WORK_ORDER_SECTIONS: WorkOrderSectionDefinition[] = [
  {
    id: "failed",
    title: "Failed",
    description: "Open work orders with a failed line step.",
    statuses: ["failed"],
    tone: "attention",
  },
  {
    id: "running",
    title: "Running",
    description: "Work orders executing on a line.",
    statuses: ["running"],
  },
  {
    id: "open",
    title: "Open",
    description: "Work orders ready to dispatch or between runs.",
    statuses: ["open"],
  },
  {
    id: "ready",
    title: "Ready",
    description: "Ready to dispatch — waiting to start.",
    statuses: ["ready"],
  },
  {
    id: "draft",
    title: "Draft",
    description: "Being scoped — not yet ready.",
    statuses: ["draft"],
  },
  {
    id: "closed",
    title: "Closed",
    description: "Completed, rejected, or failed work orders.",
    statuses: ["completed", "rejected", "closedFailed"],
  },
];

export function getWorkOrderDisplayKey(order: FactoriesWorkOrder): string {
  if (order.id) {
    return order.id.slice(0, 8);
  }
  return "—";
}

export function getWorkOrderDisplayStatus(order: FactoriesWorkOrder): WorkOrderDisplayStatus {
  if (order.state === "STATE_CLOSED") {
    if (order.result === "RESULT_REJECTED") {
      return "rejected";
    }
    if (order.result === "RESULT_FAILED") {
      return "closedFailed";
    }
    return "completed";
  }

  if (order.state === "STATE_DRAFT") {
    return "draft";
  }

  if (order.state === "STATE_READY") {
    return "ready";
  }

  const executions = order.executions ?? [];
  if (hasActiveWorkOrderExecution(executions)) {
    return "running";
  }

  //
  // Only surface the "failed" display status when the latest finished
  // execution failed. A subsequent passing retry supersedes the earlier
  // failure (its finish timestamp is newer), so the pill clears without
  // needing a state transition. On top of that we fence the failure
  // against `order.stateUpdatedAt`, which only bumps on lifecycle
  // transitions (draft → ready → open, reopen, close). Failures older
  // than the last transition belong to a previous attempt and shouldn't
  // stick to a reopened order; assignee / comment / artifact writes
  // don't bump the fence, so re-assigning a failed order doesn't hide
  // the failure.
  //
  const latestFinished = latestFinishedWorkOrderExecution(executions);
  if (latestFinished?.result === "RESULT_FAILED") {
    const finishedAt = Date.parse(latestFinished.updatedAt ?? latestFinished.createdAt ?? "");
    const stateUpdatedAt = Date.parse(order.stateUpdatedAt ?? order.updatedAt ?? "");
    if (Number.isNaN(finishedAt) || Number.isNaN(stateUpdatedAt) || finishedAt >= stateUpdatedAt) {
      return "failed";
    }
  }

  return "open";
}

export function getWorkOrderDisplayStatusMeta(status: WorkOrderDisplayStatus) {
  return DISPLAY_STATUS_META[status];
}

export function getWorkOrderStatusSummary(order: FactoriesWorkOrder): string {
  return getWorkOrderDisplayStatusMeta(getWorkOrderDisplayStatus(order)).summary;
}

export function isUnassignedWorkOrder(order: FactoriesWorkOrder): boolean {
  if (order.state === "STATE_CLOSED") {
    return false;
  }
  return !order.assignees?.length;
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
  return orders.filter((order) => isAssignedToUser(order, userId));
}

export function groupWorkOrdersBySection(
  orders: FactoriesWorkOrder[],
  sections: WorkOrderSectionDefinition[] = WORK_ORDER_SECTIONS,
): Array<{ section: WorkOrderSectionDefinition; orders: FactoriesWorkOrder[] }> {
  return sections
    .map((section) => ({
      section,
      orders: orders.filter((order) => section.statuses.includes(getWorkOrderDisplayStatus(order))),
    }))
    .filter((entry) => entry.orders.length > 0);
}

//
// countActiveWorkOrders is what the "Work Orders" badge and the factory
// detail header render — the count of orders that need attention, which
// matches the default `active` status filter (draft + ready + open +
// running + failed). Closed orders (completed / rejected / closed-failed)
// are not counted.
//
export function countActiveWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => ACTIVE_DISPLAY_STATUSES.includes(getWorkOrderDisplayStatus(order))).length;
}

export type WorkOrderOwnerFilter = "all" | "mine" | "unassigned";

export type WorkOrderStatusFilter = "all" | "active" | WorkOrderDisplayStatus;

//
// Display statuses that are considered "active" — anything not yet closed.
// The `Active` pill is the default view so newly-created `draft` orders and
// dispatch-ready `ready` orders don't get hidden behind the old `Open` pill,
// which only matched STATE_OPEN.
//
const ACTIVE_DISPLAY_STATUSES: WorkOrderDisplayStatus[] = ["draft", "ready", "open", "running", "failed"];

//
// Display statuses that count as "failed" for the Failed pill. Both open
// orders with a failed line step (`failed`) and orders closed as failed
// (`closedFailed`) are surfaced together so the pill matches user intent.
//
const FAILED_DISPLAY_STATUSES: WorkOrderDisplayStatus[] = ["failed", "closedFailed"];

export function filterWorkOrdersByOwner(
  orders: FactoriesWorkOrder[],
  ownerFilter: WorkOrderOwnerFilter,
  userId?: string,
): FactoriesWorkOrder[] {
  if (ownerFilter === "all") {
    return orders;
  }

  if (ownerFilter === "unassigned") {
    return orders.filter(isUnassignedWorkOrder);
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
  if (statusFilter === "active") {
    return orders.filter((order) => ACTIVE_DISPLAY_STATUSES.includes(getWorkOrderDisplayStatus(order)));
  }
  if (statusFilter === "failed") {
    return orders.filter((order) => FAILED_DISPLAY_STATUSES.includes(getWorkOrderDisplayStatus(order)));
  }
  return orders.filter((order) => getWorkOrderDisplayStatus(order) === statusFilter);
}

export function getWorkOrderDetailDerived(order: FactoriesWorkOrder | undefined) {
  if (!order) {
    return {
      displayStatus: null,
      statusMeta: null,
      assigneeIds: [] as string[],
      assigneeNames: [] as string[],
      isOpen: false,
      isDispatchable: false,
      isClosed: false,
    };
  }

  const displayStatus = getWorkOrderDisplayStatus(order);
  const isOpen = order.state === "STATE_OPEN";
  const isReady = order.state === "STATE_READY";

  return {
    displayStatus,
    statusMeta: getWorkOrderDisplayStatusMeta(displayStatus),
    assigneeIds: (order.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id)),
    assigneeNames: (order.assignees ?? []).map((assignee) => assignee.name ?? "Unknown"),
    isOpen,
    isDispatchable: isOpen || isReady,
    isClosed: order.state === "STATE_CLOSED",
  };
}
