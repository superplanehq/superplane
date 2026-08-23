import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";

import { formatDuration, formatMinutesSecondsDuration } from "@/lib/duration";
import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { formatCostCents, formatTokenCount } from "./splitRunFormat";
import type { SplitRunPhaseStatus } from "./splitRunMocks";

export function costUsdForDisplay(order: FactoriesWorkOrder): string {
  return formatCostCents(order.totalCostCents) ?? "$0.00";
}

export function tokensLabelForDisplay(order: FactoriesWorkOrder): string {
  return formatTokenCount(order.totalTokens) ?? "0 tokens";
}

export function startedLabelForOrder(order: Pick<FactoriesWorkOrder, "createdAt">): string {
  if (!order.createdAt) {
    return "";
  }
  return formatWorkOrderDateTime(new Date(order.createdAt));
}

export function lineStatusForDisplay(status: WorkOrderDisplayStatus): SplitRunPhaseStatus {
  if (status === "running") return "running";
  if (status === "completed") return "passed";
  if (status === "waiting") return "waiting";
  if (status === "failed") return "failed";
  return "pending";
}

export function elapsedForDisplay(
  status: WorkOrderDisplayStatus,
  order?: Pick<FactoriesWorkOrder, "createdAt" | "updatedAt">,
  now = Date.now(),
): string {
  if (status === "draft") {
    return "Not started";
  }
  if (status === "waiting") {
    return "Waiting";
  }
  const start = Date.parse(order?.createdAt ?? "");
  if (!Number.isFinite(start)) {
    return status === "running" ? "Running" : "";
  }
  const end = status === "running" ? now : Date.parse(order?.updatedAt ?? "") || now;
  const label = formatDuration(Math.max(0, end - start), { precision: "second" });
  if (!label) {
    return status === "running" ? "Running" : "";
  }
  return status === "running" ? `${label} so far` : label;
}

export function durationForStatus(status: SplitRunPhaseStatus): string {
  if (status === "waiting" || status === "pending") {
    return "—";
  }
  if (status === "running") {
    return "Running";
  }
  return "—";
}

export function durationForExecution(
  execution: Pick<FactoriesWorkOrderExecution, "createdAt" | "updatedAt">,
  status: SplitRunPhaseStatus,
  now = Date.now(),
): string {
  if (status === "waiting" || status === "pending") {
    return "—";
  }
  const start = Date.parse(execution.createdAt ?? "");
  if (!Number.isFinite(start)) {
    return durationForStatus(status);
  }
  const parsedEnd = Date.parse(execution.updatedAt ?? "");
  const end = status === "running" || !Number.isFinite(parsedEnd) ? now : parsedEnd;
  const label = formatMinutesSecondsDuration(Math.max(0, end - start));
  if (!label) {
    return durationForStatus(status);
  }
  return label;
}
