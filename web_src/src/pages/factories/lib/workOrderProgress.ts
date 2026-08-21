import type { FactoriesWorkOrder } from "@/api-client";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";
import { formatWorkOrderIdentifier } from "./workspaceKey";

// Running when the dispatch is still active, or when a step is still in
// flight. Dispatch state can lag behind step executions.
function hasActiveLineDispatch(order: FactoriesWorkOrder): boolean {
  return (order.lineDispatches ?? []).some((dispatch) => {
    if (dispatch.state === "STATE_ACTIVE") {
      return true;
    }
    return (dispatch.stepExecutions ?? []).some(isActiveWorkOrderExecution);
  });
}

/**
 * Display vocabulary for the Work Orders workspace: Draft, Running, Needs
 * attention, Completed, Failed, Rejected, Canceled. The idle-open key stays
 * `waiting` so stored filters keep working. Persisted state + result
 * columns in the database stay unchanged; this file is the single mapping
 * layer.
 */
export type WorkOrderDisplayStatus =
  | "draft"
  | "running"
  | "waiting"
  | "completed"
  | "failed"
  | "rejected"
  | "cancelled";

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
    label: "Needs attention",
    filterLabel: "Needs attention",
    summary: "A person must act before this work can continue.",
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
  rejected: {
    label: "Rejected",
    filterLabel: "Rejected",
    summary: "A person rejected this work order.",
    className:
      "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]",
    dotClassName: "bg-[color:var(--status-failed-dot)]",
  },
  cancelled: {
    label: "Canceled",
    filterLabel: "Canceled",
    summary: "This work order was canceled.",
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
  "rejected",
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
    title: "Needs attention",
    description: "Work orders that wait for a human decision.",
    statuses: ["waiting"],
  },
  {
    id: "done",
    title: "Done",
    description: "Completed, failed, rejected, or canceled work.",
    statuses: ["completed", "failed", "rejected", "cancelled"],
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
      return "rejected";
    }
    if (order.result === "RESULT_FAILED") {
      return "failed";
    }
    if (
      order.result !== "RESULT_COMPLETED" &&
      (order.lineDispatches ?? []).some((dispatch) => dispatch.result === "RESULT_CANCELLED")
    ) {
      return "cancelled";
    }
    return "completed";
  }

  if (order.state === "STATE_DRAFT") {
    return "draft";
  }

  if (hasActiveLineDispatch(order)) {
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
  const owner = (order.assignees ?? [])[0];
  const ownerId = owner?.id;

  return {
    displayStatus,
    statusMeta: getWorkOrderDisplayStatusMeta(displayStatus),
    assigneeIds: ownerId ? [ownerId] : [],
    assigneeNames: ownerId ? [owner.name ?? "Unknown"] : [],
    isOpen,
    isDispatchable: isOpen || isDraft,
    isClosed: order.state === "STATE_CLOSED",
  };
}
