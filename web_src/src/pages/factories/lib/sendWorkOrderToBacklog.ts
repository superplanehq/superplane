import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderResult } from "@/api-client";

import { pullRequestState } from "./workOrderPullRequest";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

const CLEARABLE_ARTIFACT_KINDS = new Set(["markdown", "branch", "link", "file"]);

export const SEND_WORK_ORDER_TO_BACKLOG_COPY = {
  action: "Send to backlog",
  closePullRequests: "Close previous PRs",
  clearArtifacts: "Clear previous artifacts",
  success: "Task sent to Backlog.",
  error: "SuperPlane could not send this task to the Backlog.",
} as const;

export const CLOSED_STATUS_DIALOG_COPY = {
  failed: {
    title: "Failed",
    description: "These tasks closed as failed.",
    empty: "No failed tasks.",
  },
  rejected: {
    title: "Rejected",
    description: "Archive, Reject, and Stop and Close mark a task as Rejected.",
    empty: "No rejected tasks.",
  },
} as const;

export function closedWorkOrderResultForDialogStatus(
  status: Extract<WorkOrderDisplayStatus, "failed" | "rejected">,
): FactoriesWorkOrderResult {
  if (status === "failed") {
    return "RESULT_FAILED";
  }
  return "RESULT_REJECTED";
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
