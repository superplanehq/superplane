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

/**
 * An open Review pill already asks for review. Keep other attention,
 * including Needs attention and notes next to a closed or merged
 * request.
 */
export function visibleWorkOrderCardAttentionReasons(
  reasons: WorkOrderAttentionReason[],
  cardPullRequest: WorkOrderCardPullRequest | null,
): WorkOrderAttentionReason[] {
  if (!cardPullRequest || pullRequestState(cardPullRequest.pullRequest.state) !== "open") {
    return reasons;
  }
  return reasons.filter((reason) => reason !== "approval");
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
