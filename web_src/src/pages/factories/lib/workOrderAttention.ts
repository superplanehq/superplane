import { CircleCheck, CircleX, LoaderCircle, MessageCircleQuestion, Timer, type LucideIcon } from "lucide-react";

import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";

import { getWorkOrderDisplayStatus } from "./workOrderProgress";
import { presentWorkOrderStatusNotes } from "./workOrderStatusNote";

/** Why a waiting task needs a person, or why it is addressing feedback. */
export type WorkOrderAttentionReason = "approval" | "feedback" | "question" | "failed" | "stopped" | "stalled";

export const WORK_ORDER_ATTENTION_LABEL: Record<WorkOrderAttentionReason, string> = {
  approval: "Waiting for user review",
  feedback: "Addressing user feedback",
  question: "Agent question",
  failed: "Run failed",
  stopped: "Stopped",
  stalled: "Needs attention",
};

export const WORK_ORDER_ATTENTION_CHIP_CLASSNAME: Record<WorkOrderAttentionReason, string> = {
  approval: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  feedback: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-400",
  question: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-400",
  failed: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  stopped: "border-slate-500/40 bg-slate-500/15 text-slate-800 dark:text-slate-300",
  stalled: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
};

export const WORK_ORDER_ATTENTION_ICON: Record<WorkOrderAttentionReason, LucideIcon> = {
  approval: CircleCheck,
  feedback: LoaderCircle,
  question: MessageCircleQuestion,
  failed: CircleX,
  stopped: CircleX,
  stalled: Timer,
};

/**
 * Maps a task to an attention reason. Closed failed orders and
 * waiting orders with a failed latest step are Run failed. A cancelled
 * latest step is Stopped. An active PR-feedback run is Addressing user
 * feedback. A visible status note is Waiting for user review. Waiting
 * with no note is Needs attention. Other statuses return null. The
 * note body is not classified.
 */
export function getWorkOrderAttentionReason(
  order: FactoriesWorkOrder,
  options: { addressingFeedback?: boolean } = {},
): WorkOrderAttentionReason | null {
  const status = getWorkOrderDisplayStatus(order);
  if (status === "failed") {
    return "failed";
  }
  if (status !== "waiting") {
    return null;
  }

  const latest = latestExecution(order);
  if (latest?.result === "RESULT_FAILED") {
    return "failed";
  }
  if (latest?.result === "RESULT_CANCELLED") {
    return "stopped";
  }

  if (options.addressingFeedback) {
    return "feedback";
  }

  const notes = presentWorkOrderStatusNotes(order.statusNotes, status);
  if (notes.length > 0) {
    return "approval";
  }

  return "stalled";
}

function latestExecution(order: FactoriesWorkOrder): FactoriesWorkOrderExecution | null {
  let latest: FactoriesWorkOrderExecution | null = null;
  let latestMs = Number.NEGATIVE_INFINITY;

  for (const dispatch of order.lineDispatches ?? []) {
    for (const execution of dispatch.stepExecutions ?? []) {
      const at = Date.parse(execution.updatedAt ?? execution.createdAt ?? "") || 0;
      if (at >= latestMs) {
        latest = execution;
        latestMs = at;
      }
    }
  }

  return latest;
}
