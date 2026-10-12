import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { pullRequestState } from "./workOrderPullRequest";

const CLEARABLE_ARTIFACT_KINDS = new Set(["markdown", "branch", "link", "file"]);

export const SEND_WORK_ORDER_TO_BACKLOG_COPY = {
  action: "Send to backlog",
  closePullRequests: "Close previous PRs",
  declinePullRequests: "Decline pull request",
  clearArtifacts: "Clear previous artifacts",
  success: "Task sent to Backlog.",
  error: "SuperPlane could not send this task to the Backlog.",
} as const;

export function workOrderHasCloseablePullRequests(pullRequests: FactoriesFactoryPullRequest[] | undefined): boolean {
  return (pullRequests ?? []).some((pullRequest) => {
    const state = pullRequestState(pullRequest.state);
    return state === "open" || state === "draft";
  });
}

/** Bitbucket closes read as declines: label the board action accordingly. */
export function closePullRequestsLabel(pullRequests: FactoriesFactoryPullRequest[] | undefined): string {
  const closeable = (pullRequests ?? []).filter((pullRequest) => {
    const state = pullRequestState(pullRequest.state);
    return state === "open" || state === "draft";
  });
  if (closeable.length > 0 && closeable.every((pullRequest) => pullRequest.provider === "PROVIDER_BITBUCKET")) {
    return SEND_WORK_ORDER_TO_BACKLOG_COPY.declinePullRequests;
  }
  return SEND_WORK_ORDER_TO_BACKLOG_COPY.closePullRequests;
}

export function workOrderHasClearableArtifacts(
  artifacts: Array<Pick<FactoriesWorkOrderArtifact, "type">> | undefined,
): boolean {
  return (artifacts ?? []).some((artifact) => {
    const kind = (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase();
    return CLEARABLE_ARTIFACT_KINDS.has(kind);
  });
}
