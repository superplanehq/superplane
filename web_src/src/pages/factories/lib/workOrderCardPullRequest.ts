import type { FactoriesFactoryPullRequest } from "@/api-client";

import type { WorkOrderAttentionReason } from "./workOrderAttention";
import { pullRequestLabel, pullRequestState, type FactoryPullRequestState } from "./workOrderPullRequest";

const STATE_RANK: Record<FactoryPullRequestState, number> = {
  open: 0,
  draft: 1,
  merged: 2,
  closed: 3,
};

const STATE_VERB: Record<FactoryPullRequestState, string> = {
  open: "Review",
  draft: "Draft",
  merged: "Merged",
  closed: "Closed",
};

export interface WorkOrderCardPullRequest {
  pullRequest: FactoriesFactoryPullRequest;
  extraCount: number;
}

/**
 * Picks the pull request a task card should show. Open requests win, then
 * drafts, then merged, then closed. A higher number wins ties. Extra
 * requests become a +N count on the pill.
 */
export function selectWorkOrderCardPullRequest(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
  workOrderId: string,
): WorkOrderCardPullRequest | null {
  const attached = pullRequestsForWorkOrder(pullRequests, workOrderId);
  if (attached.length === 0) {
    return null;
  }
  const sorted = [...attached].sort(compareCardPullRequests);
  return { pullRequest: sorted[0], extraCount: sorted.length - 1 };
}

export function workOrderCardPullRequestVisibleLabel(pullRequest: FactoriesFactoryPullRequest, extraCount = 0): string {
  const verb = STATE_VERB[pullRequestState(pullRequest.state)];
  const number = pullRequestLabel(pullRequest);
  const extra = extraCount > 0 ? ` +${extraCount}` : "";
  return `${verb} ${number}${extra}`;
}

export function workOrderCardPullRequestIsMergeable(pullRequest: FactoriesFactoryPullRequest | undefined): boolean {
  return Boolean(pullRequest?.mergeable) && pullRequestState(pullRequest?.state) === "open";
}

/**
 * An open Review pill already asks for review. A merged request is
 * already done, so hide the review wait until the task closes. Keep
 * other attention, including Needs attention, next to a closed
 * request. A Mergeable pill already says checks passed.
 */
export function visibleWorkOrderCardAttentionReasons(
  reasons: WorkOrderAttentionReason[],
  cardPullRequest: WorkOrderCardPullRequest | null,
): WorkOrderAttentionReason[] {
  if (!cardPullRequest) {
    return reasons;
  }
  const state = pullRequestState(cardPullRequest.pullRequest.state);
  if (state !== "open" && state !== "merged") {
    return reasons;
  }
  return reasons.filter((reason) => {
    if (reason === "approval") {
      return false;
    }
    if (reason === "checksPassed" && workOrderCardPullRequestIsMergeable(cardPullRequest.pullRequest)) {
      return false;
    }
    return true;
  });
}

export function workOrderCardPullRequestAriaLabel(pullRequest: FactoriesFactoryPullRequest, extraCount = 0): string {
  const verb = STATE_VERB[pullRequestState(pullRequest.state)];
  const number = pullRequestLabel(pullRequest);
  const extra = extraCount === 1 ? " 1 more pull request." : extraCount > 1 ? ` ${extraCount} more pull requests.` : "";
  return `${verb} pull request ${number}.${extra}`;
}

function pullRequestsForWorkOrder(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
  workOrderId: string,
): FactoriesFactoryPullRequest[] {
  if (!pullRequests || pullRequests.length === 0 || !workOrderId) {
    return [];
  }
  return pullRequests.filter((pullRequest) => pullRequest.workOrderId?.trim() === workOrderId);
}

function compareCardPullRequests(a: FactoriesFactoryPullRequest, b: FactoriesFactoryPullRequest): number {
  const rank = STATE_RANK[pullRequestState(a.state)] - STATE_RANK[pullRequestState(b.state)];
  if (rank !== 0) {
    return rank;
  }
  const number = pullRequestNumber(b) - pullRequestNumber(a);
  if (number !== 0) {
    return number;
  }
  return Date.parse(b.createdAt ?? "") - Date.parse(a.createdAt ?? "");
}

function pullRequestNumber(pullRequest: FactoriesFactoryPullRequest): number {
  const parsed = Number.parseInt(String(pullRequest.number ?? "").replace(/^#/, ""), 10);
  return Number.isFinite(parsed) ? parsed : 0;
}
