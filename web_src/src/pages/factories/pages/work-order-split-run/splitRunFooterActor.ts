import type { FactoriesWorkOrderEvent } from "@/api-client";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";

import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";

export type SplitRunFooterCloser = {
  actor?: OrgUserDisplay;
  automationName?: string;
};

interface StatusEventPayload {
  user?: { id?: string };
  automation?: { nodeName?: string; appName?: string };
  toState?: string;
  toResult?: string;
}

const CLOSED_RESULT_FOR_STATUS: Partial<Record<WorkOrderDisplayStatus, string>> = {
  completed: "completed",
  rejected: "rejected",
  failed: "failed",
  cancelled: "cancelled",
};

/**
 * Person or automation that closed or stopped this work order, taken from
 * the latest matching `order.status.updated` event. Empty when that event
 * has no user and no automation name — callers keep the "A person" copy.
 */
export function footerCloserFromEvents(
  events: FactoriesWorkOrderEvent[],
  displayStatus: WorkOrderDisplayStatus,
  resolveUser: OrgUserDisplayLookup,
): SplitRunFooterCloser {
  const payload = latestMatchingStatusPayload(events, displayStatus);
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

function latestMatchingStatusPayload(
  events: FactoriesWorkOrderEvent[],
  displayStatus: WorkOrderDisplayStatus,
): StatusEventPayload | undefined {
  const sorted = [...events].sort(compareEventsNewestFirst);
  for (const event of sorted) {
    if (event.type !== "order.status.updated") {
      continue;
    }
    const payload = (event.event ?? {}) as StatusEventPayload;
    if (statusEventMatchesFooter(payload, displayStatus)) {
      return payload;
    }
  }
  return undefined;
}

function statusEventMatchesFooter(payload: StatusEventPayload, displayStatus: WorkOrderDisplayStatus): boolean {
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
