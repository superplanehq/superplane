import type { FactoriesFactoryPullRequest } from "@/api-client";
import { humanReviewDecisionLabel, reviewTargetsPreviousRevision } from "../../../lib/humanReview";
import type { SplitRunPhase } from "../splitRunMocks";
import type { StepActionRow } from "./consoleStepActions";

export function humanReviewActionRows(
  phases: SplitRunPhase[],
  runIds: ReadonlySet<string>,
  assessmentPath: (canvasId: string, runId: string) => string,
): StepActionRow[] {
  return phases.flatMap((phase) => {
    const pullRequest = phase.pullRequestActivity?.pullRequest;
    const review = pullRequest?.humanReview;
    if (!phase.runId || !runIds.has(phase.runId) || !pullRequest || !review || review.runId !== phase.runId) {
      return [];
    }
    if (reviewTargetsPreviousRevision(pullRequest)) return [];
    const runId = phase.runId;
    const links = reviewEvidenceLinks(pullRequest, assessmentPath);
    const decision: StepActionRow = {
      id: `human-review-${phase.runId}`,
      runId,
      title: `Human review: ${humanReviewDecisionLabel(review)}`,
      detail: [review.blockingReason, review.rationale].filter(Boolean).join("\n\n") || undefined,
      links,
    };
    const reviewers = (review.selectedReviewers ?? []).map((reviewer) => ({
      id: `human-review-${phase.runId}-${reviewer.id ?? reviewer.login}`,
      runId,
      title: `Selected @${reviewer.login}`,
      detail: reviewer.reason,
    }));
    if (!review.publicationError) return [decision, ...reviewers];
    return [
      decision,
      ...reviewers,
      {
        id: `human-review-publication-${phase.runId}`,
        runId,
        title: "GitHub review update failed",
        detail: "SuperPlane will retry.",
      },
    ];
  });
}

function reviewEvidenceLinks(
  pullRequest: FactoriesFactoryPullRequest,
  assessmentPath: (canvasId: string, runId: string) => string,
): NonNullable<StepActionRow["links"]> {
  const sourceRun = [...(pullRequest.activities ?? []), ...(pullRequest.runs ?? [])].find(
    (activity) => activity.run?.id === pullRequest.humanReview?.sourceAssessmentRunId,
  )?.run;
  const links: NonNullable<StepActionRow["links"]> = [];
  if (sourceRun?.canvasId && sourceRun.id) {
    links.push({ label: "View assessment", href: assessmentPath(sourceRun.canvasId, sourceRun.id) });
  }
  if (pullRequest.humanReview?.checkUrl)
    links.push({ label: "View GitHub check", href: pullRequest.humanReview.checkUrl, external: true });
  return links;
}
