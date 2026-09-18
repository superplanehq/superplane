import { Ellipsis, ExternalLink, GitPullRequest } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import type { SplitRunDecisionTone, SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";
import {
  PULL_REQUEST_REVIEW_COPY,
  pullRequestReviewNote,
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
  compact = false,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
  compact?: boolean;
}) {
  if (compact) {
    return <CompactPullRequestReviewNote ctaLabel={ctaLabel} pullRequest={pullRequest} />;
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
          <h3 className="text-[18px] font-semibold leading-6 tracking-[-0.02em] text-foreground">
            {PULL_REQUEST_REVIEW_COPY.headline}
          </h3>
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
            <p className="text-[13px] leading-5 text-foreground/70">{PULL_REQUEST_REVIEW_COPY.closing}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

function CompactPullRequestReviewNote({
  ctaLabel,
  pullRequest,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
}) {
  return (
    <div
      className="rounded-lg border border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] p-4"
      data-testid="split-run-attention-note"
      data-variant="pull-request"
    >
      <div className="min-w-0">
        <h3 className="text-[14px] font-semibold leading-5 text-foreground">{PULL_REQUEST_REVIEW_COPY.headline}</h3>
        <p className="mt-1 text-[12px] leading-4 text-foreground/70">{PULL_REQUEST_REVIEW_COPY.closing}</p>
        <Button
          asChild
          size="sm"
          className="mt-3 bg-emerald-600 font-semibold text-white shadow-sm hover:bg-emerald-700"
        >
          <a href={pullRequest.href} target="_blank" rel="noreferrer" data-testid="split-run-pull-request-cta">
            {ctaLabel}
            <ExternalLink className="size-3.5" aria-hidden />
          </a>
        </Button>
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
  actionsOnly = false,
  onAction,
}: {
  note: SplitRunFooterNote;
  tone: SplitRunDecisionTone;
  actions: SplitRunFooterAction[];
  actionBusy: boolean;
  compact?: boolean;
  actionsOnly?: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const pullRequest = tone === "waiting" && note.cta ? pullRequestReviewNote(note) : undefined;
  if (!pullRequest || !note.cta) {
    return null;
  }
  if (actionsOnly) {
    return <MoreActionsMenu actions={actions} disabled={actionBusy} onAction={onAction} />;
  }
  return <SplitRunPullRequestReviewNote ctaLabel={note.cta.label} pullRequest={pullRequest} compact={compact} />;
}
