import { CircleCheck, CircleX, MessageCircleQuestion, Timer, type LucideIcon } from "lucide-react";

import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";

import { getWorkOrderDisplayStatus } from "./workOrderProgress";

/** Why a waiting work order needs a person. Matches the Overview redesign. */
export type WorkOrderAttentionReason = "approval" | "question" | "failed" | "stalled";

export const WORK_ORDER_ATTENTION_LABEL: Record<WorkOrderAttentionReason, string> = {
  approval: "Approval needed",
  question: "Agent question",
  failed: "Run failed",
  stalled: "No progress",
};

export const WORK_ORDER_ATTENTION_CHIP_CLASSNAME: Record<WorkOrderAttentionReason, string> = {
  approval: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  question: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-400",
  failed: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  stalled: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
};

export const WORK_ORDER_ATTENTION_ICON: Record<WorkOrderAttentionReason, LucideIcon> = {
  approval: CircleCheck,
  question: MessageCircleQuestion,
  failed: CircleX,
  stalled: Timer,
};

/**
 * Maps a work order to an attention reason. Closed failed orders and
 * waiting orders with a failed latest step are Run failed. Other waiting
 * orders use status-note wording, else No progress. Other statuses return
 * null.
 */
export function getWorkOrderAttentionReason(order: FactoriesWorkOrder): WorkOrderAttentionReason | null {
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

  const noteText = (order.statusNotes ?? [])
    .flatMap((note) => [note.key, note.headline, note.body, note.kind])
    .filter((part): part is string => Boolean(part))
    .join(" ")
    .toLowerCase();

  if (/\b(question|answer)\b/.test(noteText) || noteText.includes("agent")) {
    return "question";
  }
  if (/\b(pr|pull request|review|approv)/.test(noteText)) {
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
