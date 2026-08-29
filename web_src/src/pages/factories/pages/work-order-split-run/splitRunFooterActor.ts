import type { FactoriesWorkOrderEvent } from "@/api-client";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";

import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";

export type SplitRunFooterCloser = {
  actor?: OrgUserDisplay;
  automationName?: string;
};

interface FooterActorPayload {
  user?: { id?: string };
  automation?: { nodeName?: string; appName?: string };
  toState?: string;
  toResult?: string;
  run?: { result?: string };
}

const CLOSED_RESULT_FOR_STATUS: Partial<Record<WorkOrderDisplayStatus, string>> = {
  completed: "completed",
  rejected: "rejected",
  failed: "failed",
  cancelled: "cancelled",
};

/**
 * Person or automation that closed or stopped this task. Close
 * attribution comes from `order.status.updated`. Stop attribution comes
 * from a cancelled `step.execution.finished` (cancelling a run leaves the
 * order open and does not write a status event). Empty when that event
 * has no user and no automation name — callers keep the "A person" copy.
 */
export function footerCloserFromEvents(
  events: FactoriesWorkOrderEvent[],
  displayStatus: WorkOrderDisplayStatus,
  resolveUser: OrgUserDisplayLookup,
): SplitRunFooterCloser {
  const payload = latestMatchingActorPayload(events, displayStatus);
  if (!payload) {
    return {};
  }

  const actor = resolveUser(payload.user?.id) ?? undefined;
  const automationName = payload.automation?.nodeName?.trim() || payload.automation?.appName?.trim() || undefined;
  return {
    ...(actor ? { actor } : {}),
    ...(automationName ? { automationName } : {}),
  };
}

function latestMatchingActorPayload(
  events: FactoriesWorkOrderEvent[],
  displayStatus: WorkOrderDisplayStatus,
): FooterActorPayload | undefined {
  const sorted = [...events].sort(compareEventsNewestFirst);
  const stoppedOpen = displayStatus === "waiting" || displayStatus === "running";
  for (const event of sorted) {
    const payload = (event.event ?? {}) as FooterActorPayload;
    if (stoppedOpen && isCancelledStepFinished(event, payload)) {
      return payload;
    }
    if (event.type === "order.status.updated" && statusEventMatchesFooter(payload, displayStatus)) {
      return payload;
    }
  }
  return undefined;
}

function isCancelledStepFinished(event: FactoriesWorkOrderEvent, payload: FooterActorPayload): boolean {
  if (event.type !== "step.execution.finished") {
    return false;
  }
  return (payload.run?.result ?? "").toLowerCase() === "cancelled";
}

function statusEventMatchesFooter(payload: FooterActorPayload, displayStatus: WorkOrderDisplayStatus): boolean {
  const result = (payload.toResult ?? "").toLowerCase();
  if (displayStatus === "waiting" || displayStatus === "running") {
    return result === "cancelled";
  }
  if (payload.toState !== "closed") {
    return false;
  }
  const wanted = CLOSED_RESULT_FOR_STATUS[displayStatus];
  if (displayStatus === "completed") {
    return result === "completed" || result === "";
  }
  return Boolean(wanted) && result === wanted;
}

function compareEventsNewestFirst(left: FactoriesWorkOrderEvent, right: FactoriesWorkOrderEvent): number {
  return Date.parse(right.timestamp ?? "") - Date.parse(left.timestamp ?? "");
}
