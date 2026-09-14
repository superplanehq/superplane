import { ChevronDown, ExternalLink, GitPullRequest } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import type { SplitRunFooterAction } from "./splitRunFooter";
import { PULL_REQUEST_REVIEW_COPY, type PullRequestReviewTarget } from "./splitRunPullRequestReview";

const MARK_CLASSNAME = "flex shrink-0 items-center justify-center rounded-full bg-emerald-600 text-white";

/**
 * Decision strip for a task whose pull request is open. Header and More
 * sit on one row. The three next steps use the column width. Close
 * actions stay behind More so they do not compete with review.
 */
export function SplitRunPullRequestReviewNote({
  ctaLabel,
  pullRequest,
  actions = [],
  actionBusy = false,
  inColumn = false,
  onAction,
}: {
  ctaLabel: string;
  pullRequest: PullRequestReviewTarget;
  actions?: SplitRunFooterAction[];
  actionBusy?: boolean;
  inColumn?: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  return (
    <div
      className={cn(
        "bg-[color:var(--status-completed-bg)] px-4 py-4",
        !inColumn && "border-t border-[color:var(--status-completed-border)]",
      )}
      data-testid="split-run-attention-note"
      data-variant="pull-request"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className={`${MARK_CLASSNAME} size-8`} aria-hidden>
            <GitPullRequest className="size-4" />
          </span>
          <h3 className="min-w-0 text-[15px] font-semibold leading-5 tracking-[-0.02em] text-foreground">
            {PULL_REQUEST_REVIEW_COPY.headline}
          </h3>
        </div>
        <MoreActionsMenu actions={actions} disabled={actionBusy} onAction={onAction} />
      </div>

      <ReviewSteps inColumn={inColumn} />

      <div className={cn("mt-3 flex gap-3", inColumn ? "flex-col" : "flex-wrap items-center")}>
        <Button
          asChild
          size="lg"
          className={cn(
            "h-10 bg-emerald-600 px-5 text-[14px] font-semibold text-white shadow-sm hover:bg-emerald-700",
            inColumn && "w-full",
          )}
        >
          <a href={pullRequest.href} target="_blank" rel="noreferrer" data-testid="split-run-pull-request-cta">
            <GitPullRequest className="size-4" aria-hidden />
            {ctaLabel}
            <ExternalLink className="size-3.5" aria-hidden />
          </a>
        </Button>
        <p className="text-[12px] leading-5 text-foreground/70">{PULL_REQUEST_REVIEW_COPY.closing}</p>
      </div>
    </div>
  );
}

function ReviewSteps({ inColumn }: { inColumn: boolean }) {
  return (
    <ol
      aria-label={PULL_REQUEST_REVIEW_COPY.stepsLabel}
      className={cn(
        "mt-3 grid w-full gap-1.5",
        inColumn ? "grid-cols-1 pl-4" : "grid-cols-1 sm:grid-cols-3",
      )}
    >
      {PULL_REQUEST_REVIEW_COPY.steps.map((step, index) => (
        <li
          key={step.title}
          className="flex w-full min-w-0 items-start gap-2.5"
        >
          <span className={`${MARK_CLASSNAME} mt-0.5 size-5 text-[11px] font-semibold`} aria-hidden>
            {index + 1}
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-[13px] font-semibold leading-5 text-foreground">{step.title}</p>
            <p className="mt-0.5 text-[13px] leading-5 text-foreground/70">{step.text}</p>
          </div>
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
