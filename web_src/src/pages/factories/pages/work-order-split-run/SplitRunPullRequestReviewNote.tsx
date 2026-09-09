import { ChevronDown, ExternalLink, GitPullRequest } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import type { SplitRunFooterAction } from "./splitRunFooter";
import { PULL_REQUEST_REVIEW_COPY, type PullRequestReviewTarget } from "./splitRunPullRequestReview";

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
  actions = [],
  actionBusy = false,
  onAction,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
  actions?: SplitRunFooterAction[];
  actionBusy?: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
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
          <ReviewSteps />
          <div className="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2">
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

        <MoreActionsMenu actions={actions} disabled={actionBusy} onAction={onAction} />
      </div>
    </div>
  );
}

function ReviewSteps() {
  return (
    <ol aria-label={PULL_REQUEST_REVIEW_COPY.stepsLabel} className="mt-3.5 flex flex-col gap-2">
      {PULL_REQUEST_REVIEW_COPY.steps.map((step, index) => (
        <li key={step.title} className="flex items-start gap-2.5">
          <span className={`${MARK_CLASSNAME} mt-px size-6 text-[12px] font-semibold`} aria-hidden>
            {index + 1}
          </span>
          <p className="min-w-0 text-[14px] leading-6">
            <span className="font-semibold text-foreground">{step.title}</span>
            <span className="text-foreground/70"> {step.text}</span>
          </p>
        </li>
      ))}
    </ol>
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
          size="sm"
          className="shrink-0 text-foreground/70"
          aria-label={PULL_REQUEST_REVIEW_COPY.moreActions}
          disabled={disabled}
          data-testid="split-run-more-actions"
        >
          {PULL_REQUEST_REVIEW_COPY.more}
          <ChevronDown className="size-3.5" aria-hidden />
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
