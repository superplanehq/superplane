import type { FactoriesFactoryPullRequest } from "@/api-client";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { GitMerge, GitPullRequest, GitPullRequestClosed, GitPullRequestDraft, type LucideIcon } from "lucide-react";

import {
  workOrderCardPullRequestAriaLabel,
  workOrderCardPullRequestVisibleLabel,
} from "../lib/workOrderCardPullRequest";
import { pullRequestState, type FactoryPullRequestState } from "../lib/workOrderPullRequest";

const PR_CHIP_CLASSNAME: Record<FactoryPullRequestState, string> = {
  open: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  draft: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
  merged: "border-purple-500/30 bg-purple-500/10 text-purple-700 dark:text-purple-400",
  closed: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
};

const PR_CHIP_ICON: Record<FactoryPullRequestState, LucideIcon> = {
  open: GitPullRequest,
  draft: GitPullRequestDraft,
  merged: GitMerge,
  closed: GitPullRequestClosed,
};

/**
 * Compact pill for a pull request attached to a task. Open requests use
 * Review plus the number. The pill opens the pull request and does not
 * open the task card.
 */
export function WorkOrderPullRequestChip({
  pullRequest,
  extraCount = 0,
}: {
  pullRequest: FactoriesFactoryPullRequest;
  extraCount?: number;
}) {
  const state = pullRequestState(pullRequest.state);
  const Icon = PR_CHIP_ICON[state];
  const visible = workOrderCardPullRequestVisibleLabel(pullRequest, extraCount);
  const ariaLabel = workOrderCardPullRequestAriaLabel(pullRequest, extraCount);
  const title = pullRequest.title?.trim() || ariaLabel;
  const href = safeExternalUrl(pullRequest.url);
  const className = cn(
    "inline-flex max-w-full items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-medium",
    PR_CHIP_CLASSNAME[state],
  );
  const body = (
    <>
      <Icon className="size-3 shrink-0" aria-hidden />
      <span className="truncate">{visible}</span>
    </>
  );

  return (
    <div className="pointer-events-auto relative z-10 max-w-full" onClick={(event) => event.stopPropagation()}>
      {href ? (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className={className}
          aria-label={ariaLabel}
          title={title}
        >
          {body}
        </a>
      ) : (
        <span className={className} title={title}>
          {body}
        </span>
      )}
    </div>
  );
}
