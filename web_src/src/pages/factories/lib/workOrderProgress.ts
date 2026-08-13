import type { FactoriesWorkOrder } from "@/api-client";
import { hasActiveWorkOrderExecution } from "./workOrderExecutions";
import { formatWorkOrderIdentifier } from "./workspaceKey";

/**
 * Display vocabulary for the Work Orders workspace. Mirrors the six labels
 * from the reference application: Draft, Running, Waiting, Completed,
 * Failed, Cancelled. Persisted state + result columns in the database stay
 * unchanged; this file is the single mapping layer.
 */
export type WorkOrderDisplayStatus = "draft" | "running" | "waiting" | "completed" | "failed" | "cancelled";

const DISPLAY_STATUS_META: Record<
  WorkOrderDisplayStatus,
  { label: string; filterLabel: string; summary: string; className: string; dotClassName: string }
> = {
  draft: {
    label: "Draft",
    filterLabel: "Draft",
    summary: "Being scoped — not yet dispatched.",
    className:
      "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)] text-[color:var(--status-draft-fg)]",
    dotClassName: "bg-[color:var(--status-draft-dot)]",
  },
  running: {
    label: "Running",
    filterLabel: "Running",
    summary: "Line execution in progress.",
    className:
      "border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]",
    dotClassName: "bg-[color:var(--status-running-dot)]",
  },
  waiting: {
    label: "Waiting",
    filterLabel: "Waiting",
    summary: "Waiting for review or the next dispatch.",
    className:
      "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)] text-[color:var(--status-waiting-fg)]",
    dotClassName: "bg-[color:var(--status-waiting-dot)]",
  },
  completed: {
    label: "Completed",
    filterLabel: "Completed",
    summary: "Work order completed successfully.",
    className:
      "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] text-[color:var(--status-completed-fg)]",
    dotClassName: "bg-[color:var(--status-completed-dot)]",
  },
  failed: {
    label: "Failed",
    filterLabel: "Failed",
    summary: "Closed as failed. Line execution did not pass.",
    className:
      "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]",
    dotClassName: "bg-[color:var(--status-failed-dot)]",
  },
  cancelled: {
    label: "Cancelled",
    filterLabel: "Cancelled",
    summary: "Closed as cancelled.",
    className:
      "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)] text-[color:var(--status-cancelled-fg)]",
    dotClassName: "bg-[color:var(--status-cancelled-dot)]",
  },
};

export const WORK_ORDER_DISPLAY_STATUSES: WorkOrderDisplayStatus[] = [
  "draft",
  "running",
  "waiting",
  "completed",
  "failed",
  "cancelled",
];

/** Board lanes used across every layout. Order matters — Board renders them left-to-right. */
export type WorkOrderBoardLaneId = "backlog" | "running" | "review" | "done";

export interface WorkOrderBoardLaneDefinition {
  id: WorkOrderBoardLaneId;
  title: string;
  description: string;
  statuses: WorkOrderDisplayStatus[];
}

export const WORK_ORDER_BOARD_LANES: WorkOrderBoardLaneDefinition[] = [
  {
    id: "backlog",
    title: "Backlog",
    description: "Being scoped — not yet dispatched.",
    statuses: ["draft"],
  },
  {
    id: "running",
    title: "Running",
    description: "Executing on a line right now.",
    statuses: ["running"],
  },
  {
    id: "review",
    title: "Review",
    description: "Waiting for reviewers or the next dispatch.",
    statuses: ["waiting"],
  },
  {
    id: "done",
    title: "Done",
    description: "Completed, failed, or cancelled work.",
    statuses: ["completed", "failed", "cancelled"],
  },
];

/**
 * Human-visible identifier: prefer the workspace-scoped `SP-42` key when
 * the backend provides one, otherwise fall back to the shortened UUID we
 * used before the workspace-key rollout.
 *
 * `factoryKey` is the parent factory's key; passing it lets callers render
 * an identifier when only the immutable per-order `number` is available.
 */
export function getWorkOrderDisplayKey(order: FactoriesWorkOrder, factoryKey?: string | null): string {
  if (order.key) {
    return order.key;
  }
  const composed = formatWorkOrderIdentifier(factoryKey ?? undefined, order.number ?? undefined);
  if (composed) {
    return composed;
  }
  if (order.id) {
    return order.id.slice(0, 8);
  }
  return "—";
}

export function getWorkOrderDisplayStatus(order: FactoriesWorkOrder): WorkOrderDisplayStatus {
  if (order.state === "STATE_CLOSED") {
    if (order.result === "RESULT_REJECTED") {
      return "cancelled";
    }
    if (order.result === "RESULT_FAILED") {
      return "failed";
    }
    return "completed";
  }

  if (order.state === "STATE_DRAFT") {
    return "draft";
  }

  if (hasActiveWorkOrderExecution(order.executions ?? [])) {
    return "running";
  }

  return "waiting";
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

// Active covers everything that is not yet closed. Kept for the badge in
// the workspace navigation.
const ACTIVE_DISPLAY_STATUSES: WorkOrderDisplayStatus[] = ["draft", "running", "waiting"];

export function countActiveWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => ACTIVE_DISPLAY_STATUSES.includes(getWorkOrderDisplayStatus(order))).length;
}

export type WorkOrderOwnerFilter = "all" | "mine" | "unassigned";
export type WorkOrderStatusFilter = "all" | "active" | WorkOrderDisplayStatus;

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
  return orders.filter((order) => getWorkOrderDisplayStatus(order) === statusFilter);
}

export function groupWorkOrdersByLane(
  orders: FactoriesWorkOrder[],
  lanes: WorkOrderBoardLaneDefinition[] = WORK_ORDER_BOARD_LANES,
): Array<{ lane: WorkOrderBoardLaneDefinition; orders: FactoriesWorkOrder[] }> {
  return lanes.map((lane) => ({
    lane,
    orders: orders.filter((order) => lane.statuses.includes(getWorkOrderDisplayStatus(order))),
  }));
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
