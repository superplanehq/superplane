import { ExternalLink } from "lucide-react";

import type { FactoriesFactoryPullRequest } from "@/api-client";
import { Button } from "@/components/ui/button";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { pullRequestRepositoryLabel } from "../../lib/workOrderPullRequest";
import { PullRequestConfidence } from "./redesign/consoleCheckRows";
import { SplitRunPullRequestMergeAction } from "./SplitRunPullRequestMergeAction";
import { multiplePullRequestReviewCopy } from "./splitRunPullRequestReview";

/**
 * A task with pull requests in several repositories: one compact row per
 * pull request, each with its repository, title, and own review and merge
 * actions. Numbers repeat across repositories, so the row names the repository.
 */
export function MultiPullRequestReviewNote({
  pullRequests,
  organizationId,
  factoryId,
  orderId,
  canAct,
  stacked,
  continueFixes,
  actionBusy,
}: {
  pullRequests: FactoriesFactoryPullRequest[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canAct: boolean;
  stacked: boolean;
  continueFixes?: () => void;
  actionBusy: boolean;
}) {
  const copy = multiplePullRequestReviewCopy(pullRequests.length);
  return (
    <div
      className={
        stacked
          ? undefined
          : "rounded-lg border border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] p-4"
      }
      data-testid="split-run-attention-note"
      data-variant="pull-request"
    >
      <h3 className="text-[14px] font-semibold leading-5 text-foreground">{copy.headline}</h3>
      <p className="mt-1 text-[12px] leading-4 text-foreground/70">{copy.closing}</p>
      <ul aria-label="Pull requests" className="mt-3 flex flex-col divide-y divide-foreground/10">
        {pullRequests.map((pullRequest, index) => (
          <li key={pullRequest.id ?? pullRequest.url ?? index} className="py-2.5 first:pt-0 last:pb-0">
            <ReviewPullRequestRow
              pullRequest={pullRequest}
              organizationId={organizationId}
              factoryId={factoryId}
              orderId={orderId}
              canAct={canAct}
            />
          </li>
        ))}
      </ul>
      {continueFixes ? (
        <div className="mt-3">
          <ContinueFixesButton onContinue={continueFixes} disabled={actionBusy} />
        </div>
      ) : null}
    </div>
  );
}

function ReviewPullRequestRow({
  pullRequest,
  organizationId,
  factoryId,
  orderId,
  canAct,
}: {
  pullRequest: FactoriesFactoryPullRequest;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canAct: boolean;
}) {
  const href = safeExternalUrl(pullRequest.url);
  const label = pullRequestRepositoryLabel(pullRequest);
  const title = pullRequest.title?.trim();
  return (
    <div className="flex min-w-0 flex-col gap-2" data-testid="split-run-review-pull-request">
      <div className="min-w-0">
        <p className="truncate text-[12px] leading-4 text-muted-foreground">{label}</p>
        {title ? <p className="line-clamp-2 text-[13px] font-medium leading-5 text-foreground">{title}</p> : null}
      </div>
      <PullRequestConfidence repository={pullRequest.repository} />
      <div className="flex flex-wrap items-center gap-2">
        {href ? (
          <Button asChild size="sm" className="bg-emerald-600 font-semibold text-white shadow-sm hover:bg-emerald-700">
            <a
              href={href}
              target="_blank"
              rel="noreferrer"
              aria-label={`Review ${label}`}
              data-testid="split-run-pull-request-cta"
            >
              Review
              <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </Button>
        ) : null}
        <SplitRunPullRequestMergeAction
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          pullRequest={pullRequest}
          canAct={canAct}
          compact
        />
      </div>
    </div>
  );
}

/** Resumes paused automatic fixes. Renders nothing when fixes are not paused. */
export function ContinueFixesButton({
  onContinue,
  disabled,
  size = "sm",
}: {
  onContinue?: () => void;
  disabled: boolean;
  size?: "sm" | "lg";
}) {
  if (!onContinue) {
    return null;
  }
  return (
    <Button
      type="button"
      size={size}
      disabled={disabled}
      onClick={onContinue}
      data-testid="split-run-footer-continue-fixes"
    >
      Continue fixes
    </Button>
  );
}
