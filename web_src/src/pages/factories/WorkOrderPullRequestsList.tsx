import type { FactoriesFactoryPullRequest } from "@/api-client";
import { useRevealAfterPending } from "@/hooks/useRevealAfterPending";
import { cn } from "@/lib/utils";

import { LOADING_REVEAL_CLASSNAME } from "./lib/loadingReveal";
import { WorkOrderListSkeleton } from "./WorkOrderListSkeleton";
import { WorkOrderPullRequestInline } from "./WorkOrderPullRequestInline";

interface WorkOrderPullRequestsListProps {
  pullRequests: FactoriesFactoryPullRequest[];
  isLoading: boolean;
  error?: Error | null;
}

export function WorkOrderPullRequestsList({ pullRequests, isLoading, error }: WorkOrderPullRequestsListProps) {
  const reveal = useRevealAfterPending(isLoading);
  return (
    <section>
      <h3 className="workspace-section-label">Pull requests</h3>

      <div className="mt-2">
        {error ? (
          <p className="text-[13px] text-destructive">Failed to load pull requests.</p>
        ) : isLoading ? (
          <WorkOrderListSkeleton label="Loading pull requests" />
        ) : pullRequests.length === 0 ? (
          <p className={cn("text-[13px] text-muted-foreground", reveal && LOADING_REVEAL_CLASSNAME)}>
            No pull requests yet.
          </p>
        ) : (
          <ul className={cn(reveal && LOADING_REVEAL_CLASSNAME)} data-reveal={reveal ? "" : undefined}>
            {pullRequests.map((pullRequest) => (
              <li
                className="flex items-center py-1.5"
                key={pullRequest.id ?? `${pullRequest.url}-${pullRequest.number}`}
              >
                <WorkOrderPullRequestInline className="w-full justify-start" pullRequest={pullRequest} showTitle />
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
