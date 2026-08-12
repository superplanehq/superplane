import type { FactoriesWorkOrder } from "@/api-client";
import { hasActiveWorkOrderExecution, latestFinishedWorkOrderExecution } from "./workOrderExecutions";

export type WorkOrderDisplayStatus =
  | "draft"
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
    summary: "Being scoped — not yet dispatched",
    className: "border-border bg-accent/50 text-muted-foreground",
  },
  open: {
    label: "Open",
    filterLabel: "Open",
    summary: "Ready to dispatch or between runs",
    className: "border-border bg-accent/50 text-foreground",
  },
  running: {
    label: "Running",
    filterLabel: "Running",
    summary: "Line execution in progress",
    className: "border-border bg-accent/50 text-[color:var(--status-running)]",
  },
  failed: {
    label: "Failed",
    filterLabel: "Failed",
    summary: "A line step failed",
    className: "border-border bg-accent/50 text-[color:var(--status-danger)]",
  },
  completed: {
    label: "Completed",
    filterLabel: "Completed",
    summary: "Work order completed",
    className: "border-border bg-accent/50 text-[color:var(--status-success)]",
  },
  rejected: {
    label: "Rejected",
    filterLabel: "Rejected",
    summary: "Work order rejected",
    className: "border-border bg-accent/50 text-muted-foreground",
  },
  closedFailed: {
    label: "Failed",
    filterLabel: "Failed",
    summary: "Work order closed as failed",
    className: "border-border bg-accent/50 text-[color:var(--status-danger)]",
  },
};

export type WorkOrderSectionId = "failed" | "running" | "open" | "draft" | "closed";

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
    id: "draft",
    title: "Draft",
    description: "Being scoped — not yet dispatched.",
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

  const executions = order.executions ?? [];
  if (hasActiveWorkOrderExecution(executions)) {
    return "running";
  }

  // "failed" only if the newest execution failed AND finished after
  // the last write to the order (reopen / reassign / comment / artifact),
  // so a fresh attempt or triage clears the pill until a new failure.
  const latestFinished = latestFinishedWorkOrderExecution(executions);
  if (latestFinished?.result === "RESULT_FAILED") {
    const finishedAt = Date.parse(latestFinished.updatedAt ?? latestFinished.createdAt ?? "");
    const orderUpdatedAt = Date.parse(order.updatedAt ?? "");
    if (Number.isNaN(finishedAt) || Number.isNaN(orderUpdatedAt) || finishedAt >= orderUpdatedAt) {
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

// countActiveWorkOrders backs the "Work Orders" badge; it matches the
// default `active` filter (draft + open + running + failed).
export function countActiveWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => ACTIVE_DISPLAY_STATUSES.includes(getWorkOrderDisplayStatus(order))).length;
}

export type WorkOrderOwnerFilter = "all" | "mine" | "unassigned";

export type WorkOrderStatusFilter = "all" | "active" | WorkOrderDisplayStatus;

// "Active" pill covers everything not yet closed (draft included, so
// new orders aren't hidden behind the STATE_OPEN-only `Open` pill).
const ACTIVE_DISPLAY_STATUSES: WorkOrderDisplayStatus[] = ["draft", "open", "running", "failed"];

// "Failed" pill unions in-flight failures with closed-as-failed orders.
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
  const isDraft = order.state === "STATE_DRAFT";

  return {
    displayStatus,
    statusMeta: getWorkOrderDisplayStatusMeta(displayStatus),
    assigneeIds: (order.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id)),
    assigneeNames: (order.assignees ?? []).map((assignee) => assignee.name ?? "Unknown"),
    isOpen,
    isDispatchable: isOpen || isDraft,
    isClosed: order.state === "STATE_CLOSED",
  };
}
