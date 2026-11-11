import type { FactoriesFactoryPullRequest } from "@/api-client";

type ReviewState = NonNullable<FactoriesFactoryPullRequest["humanReview"]>;

/** An old approval must never appear to authorize a newly observed head commit. */
export function reviewTargetsPreviousRevision(pullRequest: FactoriesFactoryPullRequest): boolean {
  const review = pullRequest.humanReview;
  const currentCommit = pullRequest.currentRevision?.sha;
  return Boolean(review && review.status !== "disabled" && currentCommit && review.assessedCommit !== currentCommit);
}

export function humanReviewDecisionLabel(review: ReviewState): string {
  switch (review.status) {
    case "waiting_assessment":
      return "Waiting for merge confidence";
    case "waiting_decision":
      return "Waiting for an agent decision";
    case "failed":
      return "Blocked";
    case "disabled":
      return "Disabled";
    case "superseded":
      return "Superseded";
    case "decided":
      if (review.blockingReason) return "Blocked";
      if (!review.reviewRequired) return "Not required";
      return review.approvalIds?.length ? "Approved" : "Approval required";
    default:
      return "Waiting for evaluation";
  }
}
