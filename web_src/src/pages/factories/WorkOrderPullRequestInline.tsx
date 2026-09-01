import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import type { FactoriesFactoryPullRequest } from "@/api-client";
import { ExternalLink, GitMerge, GitPullRequest, GitPullRequestClosed, GitPullRequestDraft } from "lucide-react";

import {
  pullRequestLabel,
  pullRequestListLabel,
  pullRequestState,
  type FactoryPullRequestState,
} from "./lib/workOrderPullRequest";

const PR_STATE_PRESENTATION: Record<FactoryPullRequestState, { icon: typeof GitPullRequest; className: string }> = {
  open: { icon: GitPullRequest, className: "text-emerald-600 dark:text-emerald-400" },
  draft: { icon: GitPullRequestDraft, className: "text-muted-foreground" },
  closed: { icon: GitPullRequestClosed, className: "text-red-600 dark:text-red-400" },
  merged: { icon: GitMerge, className: "text-purple-600 dark:text-purple-400" },
};

const pullRequestInlineClassName =
  "inline-flex min-w-0 max-w-full items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em] text-foreground";

export function WorkOrderPullRequestInline({
  pullRequest,
  className,
  showTitle = false,
}: {
  pullRequest: FactoriesFactoryPullRequest;
  className?: string;
  /** Include the PR title after the number (sidebar lists). */
  showTitle?: boolean;
}) {
  const state = pullRequestState(pullRequest.state);
  const { icon: Icon, className: iconClassName } = PR_STATE_PRESENTATION[state];
  const label = showTitle ? pullRequestListLabel(pullRequest) : pullRequestLabel(pullRequest);
  const title = pullRequest.title?.trim();
  const safeUrl = safeExternalUrl(pullRequest.url);
  const content = (
    <>
      <Icon className={cn("size-3.5 shrink-0", iconClassName)} aria-hidden />
      <span className="truncate" title={title && title !== label ? title : undefined}>
        {label}
      </span>
      {safeUrl ? <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden /> : null}
    </>
  );
  const classes = cn(pullRequestInlineClassName, className);

  if (!safeUrl) {
    return <span className={classes}>{content}</span>;
  }

  return (
    <a href={safeUrl} target="_blank" rel="noopener noreferrer" className={cn(classes, "hover:underline")}>
      {content}
    </a>
  );
}
