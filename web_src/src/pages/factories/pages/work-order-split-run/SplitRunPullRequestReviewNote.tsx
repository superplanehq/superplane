import { Ellipsis, ExternalLink, GitPullRequest } from "lucide-react";

import type { FactoriesFactoryPullRequest } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryPullRequestMergeability } from "@/hooks/useFactoryPullRequestMerge";
import { FEATURE_FACTORY_PULL_REQUEST_MERGE } from "@/lib/experimentalFeatures";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { SplitRunPullRequestMergeAction } from "./SplitRunPullRequestMergeAction";
import type { SplitRunDecisionTone, SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";
import {
  isGitHubPullRequest,
  isMergedPullRequest,
  PULL_REQUEST_REVIEW_COPY,
  pullRequestForReviewHref,
  pullRequestReviewCopy,
  pullRequestReviewNote,
  type PullRequestReviewCopy,
  type PullRequestReviewTarget,
} from "./splitRunPullRequestReview";

const MARK_CLASSNAME = "flex shrink-0 items-center justify-center rounded-full bg-emerald-600 text-white";

/**
 * Decision strip for a task whose pull request is open. One message (the
 * pull request is ready), one large call to action (review it), and a
 * three-step guide. The automation-authored headline and body are not
 * shown here; the steps say the same thing in a scannable form. Close
 * actions stay available behind More so they do not compete with review.
 */
export function SplitRunPullRequestReviewNote({
  ctaLabel,
  pullRequest,
  trackedPullRequest,
  organizationId,
  factoryId,
  orderId,
  canAct = true,
  compact = false,
  stacked = false,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
  trackedPullRequest?: FactoriesFactoryPullRequest;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canAct?: boolean;
  compact?: boolean;
  /** In a tinted panel section: no box of its own. */
  stacked?: boolean;
}) {
  const copy = usePullRequestReviewCopy(organizationId, factoryId, trackedPullRequest);
  if (compact) {
    return (
      <CompactPullRequestReviewNote
        ctaLabel={ctaLabel}
        pullRequest={pullRequest}
        trackedPullRequest={trackedPullRequest}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        canAct={canAct}
        copy={copy}
        stacked={stacked}
      />
    );
  }

  return (
    <div
      className="border-t border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] px-5 py-5"
      data-testid="split-run-attention-note"
      data-variant="pull-request"
    >
      <div className="flex items-start gap-3.5">
        <span className={`${MARK_CLASSNAME} size-10`} aria-hidden>
          <GitPullRequest className="size-5" />
        </span>

        <div className="min-w-0 flex-1">
          <h3 className="text-[18px] font-semibold leading-6 tracking-[-0.02em] text-foreground">{copy.headline}</h3>
          <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2">
            <Button
              asChild
              size="lg"
              className="h-11 bg-emerald-600 px-6 text-[15px] font-semibold text-white shadow-sm hover:bg-emerald-700"
            >
              <a href={pullRequest.href} target="_blank" rel="noreferrer" data-testid="split-run-pull-request-cta">
                <GitPullRequest className="size-[18px]" aria-hidden />
                {ctaLabel}
                <ExternalLink className="size-4" aria-hidden />
              </a>
            </Button>
            <SplitRunPullRequestMergeAction
              organizationId={organizationId}
              factoryId={factoryId}
              orderId={orderId}
              pullRequest={trackedPullRequest}
              canAct={canAct}
            />
            <p className="text-[13px] leading-5 text-foreground/70">{copy.closing}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * The checks state comes from the merge gate, so the strip can say
 * "waits for checks" instead of "ready" while checks still run. Without
 * the merge feature the gate is not fetched and the copy stays generic.
 */
function usePullRequestReviewCopy(
  organizationId: string | undefined,
  factoryId: string | undefined,
  pullRequest: FactoriesFactoryPullRequest | undefined,
): PullRequestReviewCopy {
  const { has } = useExperimentalFeature(organizationId);
  const enabled = Boolean(
    has(FEATURE_FACTORY_PULL_REQUEST_MERGE) &&
      organizationId &&
      factoryId &&
      pullRequest?.id &&
      isGitHubPullRequest(pullRequest) &&
      !isMergedPullRequest(pullRequest),
  );
  const mergeability = useFactoryPullRequestMergeability(organizationId ?? "", factoryId ?? "", pullRequest?.id ?? "", {
    enabled,
  });
  return pullRequestReviewCopy(enabled ? mergeability.data : undefined);
}

function CompactPullRequestReviewNote({
  ctaLabel,
  pullRequest,
  trackedPullRequest,
  organizationId,
  factoryId,
  orderId,
  canAct,
  copy,
  stacked,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
  trackedPullRequest?: FactoriesFactoryPullRequest;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  canAct: boolean;
  copy: PullRequestReviewCopy;
  stacked: boolean;
}) {
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
      <div className="min-w-0">
        <h3 className="text-[14px] font-semibold leading-5 text-foreground">{copy.headline}</h3>
        <p className="mt-1 text-[12px] leading-4 text-foreground/70">{copy.closing}</p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <Button asChild size="sm" className="bg-emerald-600 font-semibold text-white shadow-sm hover:bg-emerald-700">
            <a href={pullRequest.href} target="_blank" rel="noreferrer" data-testid="split-run-pull-request-cta">
              {ctaLabel}
              <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </Button>
          <SplitRunPullRequestMergeAction
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
            pullRequest={trackedPullRequest}
            canAct={canAct}
            compact
          />
        </div>
      </div>
    </div>
  );
}

function MoreActionsMenu({
  actions,
  disabled,
  onAction,
}: {
  actions: SplitRunFooterAction[];
  disabled: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  if (actions.length === 0) {
    return null;
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-foreground/70 hover:bg-slate-950/5 dark:hover:bg-white/10"
          aria-label={PULL_REQUEST_REVIEW_COPY.moreActions}
          disabled={disabled}
          data-testid="split-run-more-actions"
        >
          <Ellipsis className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-40">
        {actions.map((action) => (
          <DropdownMenuItem
            key={action.id}
            onSelect={() => onAction?.(action)}
            data-testid={`split-run-footer-${action.id}`}
          >
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function WaitingPullRequestReview({
  note,
  tone,
  actions,
  actionBusy,
  compact = false,
  stacked = false,
  actionsOnly = false,
  organizationId,
  factoryId,
  orderId,
  pullRequests,
  canAct = true,
  onAction,
}: {
  note: SplitRunFooterNote;
  tone: SplitRunDecisionTone;
  actions: SplitRunFooterAction[];
  actionBusy: boolean;
  compact?: boolean;
  /** In a tinted panel section: no box of its own. */
  stacked?: boolean;
  actionsOnly?: boolean;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  canAct?: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const pullRequest = tone === "waiting" && note.cta ? pullRequestReviewNote(note) : undefined;
  if (!pullRequest || !note.cta) {
    return null;
  }
  if (actionsOnly) {
    return <MoreActionsMenu actions={actions} disabled={actionBusy} onAction={onAction} />;
  }
  return (
    <SplitRunPullRequestReviewNote
      ctaLabel={note.cta.label}
      pullRequest={pullRequest}
      trackedPullRequest={pullRequestForReviewHref(pullRequests, pullRequest.href)}
      organizationId={organizationId}
      factoryId={factoryId}
      orderId={orderId}
      canAct={canAct}
      compact={compact}
      stacked={stacked}
    />
  );
}
