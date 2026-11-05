import type { FactoriesFactoryPullRequest } from "@/api-client";
import { Link } from "@/components/Link/link";

import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryAppRunPath } from "../lib/factoryPagePaths";

type ReviewState = NonNullable<FactoriesFactoryPullRequest["humanReview"]>;

export function HumanReviewState({ pullRequest }: { pullRequest: FactoriesFactoryPullRequest }) {
  const layout = useOptionalFactoriesLayout();
  const review = pullRequest.humanReview;
  if (!review) return null;
  if (reviewTargetsPreviousRevision(pullRequest))
    return (
      <section
        aria-label="Human review"
        className="space-y-2 rounded-lg border border-border bg-background/60 p-3 text-sm"
      >
        <h4 className="font-medium">Human review: Waiting for merge confidence</h4>
        <p className="text-muted-foreground">
          The pull request changed. The new commit needs an assessment and review decision.
        </p>
      </section>
    );
  const sourceRun = [...(pullRequest.activities ?? []), ...(pullRequest.runs ?? [])].find(
    (activity) => activity.run?.id === review.sourceAssessmentRunId,
  )?.run;
  const assessmentHref =
    layout && sourceRun?.canvasId && sourceRun.id
      ? factoryAppRunPath(layout.organizationId, layout.routeSegment, sourceRun.canvasId, sourceRun.id)
      : undefined;

  return (
    <section
      aria-label="Human review"
      className="space-y-2 rounded-lg border border-border bg-background/60 p-3 text-sm"
    >
      <h4 className="font-medium">Human review: {decisionLabel(review)}</h4>
      {review.rationale ? <p className="text-muted-foreground">{review.rationale}</p> : null}
      {review.blockingReason ? <p role="status">{review.blockingReason}</p> : null}
      <SelectedReviewers reviewers={review.selectedReviewers} />
      {review.publicationError ? <p role="alert">The GitHub check could not update. SuperPlane will retry.</p> : null}
      <ReviewEvidenceLinks assessmentHref={assessmentHref} checkUrl={review.checkUrl} />
    </section>
  );
}

function SelectedReviewers({ reviewers }: { reviewers: ReviewState["selectedReviewers"] }) {
  if (!reviewers?.length) return null;
  return (
    <ul className="space-y-1">
      {reviewers.map((reviewer) => (
        <li key={reviewer.id ?? reviewer.login}>
          <span className="font-medium">@{reviewer.login}</span>
          {reviewer.reason ? <span className="text-muted-foreground">: {reviewer.reason}</span> : null}
        </li>
      ))}
    </ul>
  );
}

function ReviewEvidenceLinks({ assessmentHref, checkUrl }: { assessmentHref?: string; checkUrl?: string }) {
  if (!assessmentHref && !checkUrl) return null;
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1">
      {assessmentHref ? (
        <Link href={assessmentHref} className="underline">
          View assessment
        </Link>
      ) : null}
      {checkUrl ? (
        <a href={checkUrl} target="_blank" rel="noreferrer" className="underline">
          View GitHub check
        </a>
      ) : null}
    </div>
  );
}

/** An old approval must never appear to authorize a newly observed head commit. */
function reviewTargetsPreviousRevision(pullRequest: FactoriesFactoryPullRequest): boolean {
  const review = pullRequest.humanReview;
  const currentCommit = pullRequest.currentRevision?.sha;
  return Boolean(review && review.status !== "disabled" && currentCommit && review.assessedCommit !== currentCommit);
}

function decisionLabel(review: ReviewState): string {
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
      return "Superseded by a newer decision";
    case "decided":
      if (review.blockingReason) return "Blocked";
      if (!review.reviewRequired) return "Approval is not required";
      return review.approvalIds?.length ? "Approved" : "Approval required";
    default:
      return "Waiting for evaluation";
  }
}
