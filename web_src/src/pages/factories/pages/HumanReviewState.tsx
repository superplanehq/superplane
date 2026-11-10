import type { FactoriesFactoryPullRequest } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronDown } from "lucide-react";

import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { humanReviewDecisionLabel, reviewTargetsPreviousRevision } from "../lib/humanReview";
import { factoryAppRunPath } from "../lib/factoryPagePaths";

type ReviewState = NonNullable<FactoriesFactoryPullRequest["humanReview"]>;

export function HumanReviewState({ pullRequest }: { pullRequest: FactoriesFactoryPullRequest }) {
  const layout = useOptionalFactoriesLayout();
  const review = pullRequest.humanReview;
  if (!review) return null;
  if (reviewTargetsPreviousRevision(pullRequest))
    return (
      <section aria-label="Human review">
        <h4 className="text-[14px] font-semibold leading-5">Human review</h4>
        <p role="status" className="text-[12px] leading-4 text-muted-foreground">
          Waiting for merge confidence
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
  return <HumanReviewDecision review={review} assessmentHref={assessmentHref} />;
}

function HumanReviewDecision({ review, assessmentHref }: { review: ReviewState; assessmentHref?: string }) {
  const summary = (
    <span className="min-w-0 flex-1">
      <span className="block text-[14px] font-semibold leading-5 text-foreground">Human review</span>
      <span className="block text-[12px] font-normal leading-4 text-muted-foreground">
        {humanReviewDecisionLabel(review)}
        {review.selectedReviewers?.length ? (
          <> · {review.selectedReviewers.map((reviewer) => `@${reviewer.login}`).join(", ")}</>
        ) : null}
      </span>
    </span>
  );

  return (
    <Collapsible asChild>
      <section aria-label="Human review">
        <h4>
          {hasReviewDetails(review, assessmentHref) ? (
            <CollapsibleTrigger className="group flex w-full items-center gap-2.5 text-left">
              {summary}
              <ChevronDown
                className="size-4 shrink-0 -rotate-90 text-muted-foreground transition-transform group-data-[state=open]:rotate-0"
                aria-hidden
              />
            </CollapsibleTrigger>
          ) : (
            summary
          )}
        </h4>
        {review.publicationError ? (
          <p role="alert" className="mt-2 text-[12px] leading-4">
            GitHub review update failed. SuperPlane will retry.
          </p>
        ) : null}
        <CollapsibleContent className="mt-2 space-y-2 border-t border-border/70 pt-2 text-[12px] leading-5">
          {review.blockingReason ? <p role="status">{review.blockingReason}</p> : null}
          {review.rationale ? <p className="text-muted-foreground">{review.rationale}</p> : null}
          <SelectedReviewers reviewers={review.selectedReviewers} />
          <ReviewEvidenceLinks assessmentHref={assessmentHref} checkUrl={review.checkUrl} />
        </CollapsibleContent>
      </section>
    </Collapsible>
  );
}

function hasReviewDetails(review: ReviewState, assessmentHref?: string): boolean {
  return Boolean(
    review.rationale ||
      review.blockingReason ||
      review.selectedReviewers?.some((reviewer) => reviewer.reason) ||
      assessmentHref ||
      review.checkUrl,
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
