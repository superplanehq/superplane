import type { FactoriesWorkOrder } from "@/api-client";
import { hasActiveWorkOrderExecution, hasFailedWorkOrderExecution } from "./workOrderExecutions";

export type WorkOrderDisplayStatus = "open" | "running" | "failed" | "completed" | "rejected";

const DISPLAY_STATUS_META: Record<
  WorkOrderDisplayStatus,
  { label: string; filterLabel: string; summary: string; className: string }
> = {
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
};

export type WorkOrderSectionId = "failed" | "running" | "open" | "closed";

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
    id: "closed",
    title: "Closed",
    description: "Completed or rejected work orders.",
    statuses: ["completed", "rejected"],
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
    return order.result === "RESULT_REJECTED" ? "rejected" : "completed";
  }

  const executions = order.executions ?? [];
  if (hasActiveWorkOrderExecution(executions)) {
    return "running";
  }

  if (hasFailedWorkOrderExecution(executions)) {
    return "failed";
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
  return order.state === "STATE_OPEN" && !order.assignees?.length;
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

export function countOpenWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => order.state === "STATE_OPEN").length;
}

export type WorkOrderOwnerFilter = "all" | "mine" | "unassigned";

export type WorkOrderStatusFilter = "all" | WorkOrderDisplayStatus;

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
    };
  }

  const displayStatus = getWorkOrderDisplayStatus(order);

  return {
    displayStatus,
    statusMeta: getWorkOrderDisplayStatusMeta(displayStatus),
    assigneeIds: (order.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id)),
    assigneeNames: (order.assignees ?? []).map((assignee) => assignee.name ?? "Unknown"),
    isOpen: order.state === "STATE_OPEN",
  };
}
