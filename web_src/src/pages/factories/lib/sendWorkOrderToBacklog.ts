import type {
  FactoriesFactoryPullRequest,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderSummary,
} from "@/api-client";

import { pullRequestState } from "./workOrderPullRequest";
import { getWorkOrderDisplayKey, getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

const CLEARABLE_ARTIFACT_KINDS = new Set(["markdown", "branch", "link", "file"]);

export const SEND_WORK_ORDER_TO_BACKLOG_COPY = {
  action: "Send to backlog",
  closePullRequests: "Close previous PRs",
  clearArtifacts: "Clear previous artifacts",
  success: "Task sent to Backlog.",
  error: "SuperPlane could not send this task to the Backlog.",
} as const;

export const CLOSED_STATUS_DIALOG_COPY = {
  title: "Closed tasks",
  description: "Failed, Rejected, and Canceled tasks. Send a task to Backlog to work on it again.",
  empty: "No closed tasks.",
  searchEmpty: "No tasks match this search.",
  searchMore: "No matching tasks on this page.",
  searchPlaceholder: "Search tasks",
} as const;

export const CLOSED_STATUS_DIALOG_RESULTS = [
  "RESULT_FAILED",
  "RESULT_REJECTED",
] as const satisfies readonly FactoriesWorkOrderResult[];

export function closedStatusEmptyLabel(searchActive: boolean, hasMorePages: boolean): string {
  if (!searchActive) {
    return CLOSED_STATUS_DIALOG_COPY.empty;
  }
  if (hasMorePages) {
    return CLOSED_STATUS_DIALOG_COPY.searchMore;
  }
  return CLOSED_STATUS_DIALOG_COPY.searchEmpty;
}

export function closedStatusTaskMatchesSearch(
  order: FactoriesWorkOrderSummary,
  factoryKey: string,
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return true;
  }
  const status = getWorkOrderDisplayStatus(order);
  const haystack = [
    order.title,
    order.key,
    order.number,
    getWorkOrderDisplayKey(order, factoryKey),
    getWorkOrderDisplayStatusMeta(status).label,
  ]
    .filter((value): value is string => Boolean(value))
    .join(" ")
    .toLowerCase();
  return haystack.includes(needle);
}

export function workOrderHasCloseablePullRequests(pullRequests: FactoriesFactoryPullRequest[] | undefined): boolean {
  return (pullRequests ?? []).some((pullRequest) => {
    const state = pullRequestState(pullRequest.state);
    return state === "open" || state === "draft";
  });
}

export function workOrderHasClearableArtifacts(
  artifacts: Array<Pick<FactoriesWorkOrderArtifact, "type">> | undefined,
): boolean {
  return (artifacts ?? []).some((artifact) => {
    const kind = (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase();
    return CLEARABLE_ARTIFACT_KINDS.has(kind);
  });
}
