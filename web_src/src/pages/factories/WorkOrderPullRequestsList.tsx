import type { FactoriesFactoryPullRequest } from "@/api-client";

import { pullRequestCreatorLabel } from "./lib/workOrderCreator";
import { WorkOrderPullRequestInline } from "./WorkOrderPullRequestInline";

interface WorkOrderPullRequestsListProps {
  pullRequests: FactoriesFactoryPullRequest[];
  isLoading: boolean;
  error?: Error | null;
}

export function WorkOrderPullRequestsList({ pullRequests, isLoading, error }: WorkOrderPullRequestsListProps) {
  return (
    <section>
      <h3 className="workspace-section-label">Pull requests</h3>

      <div className="mt-2">
        {error ? (
          <p className="text-[13px] text-destructive">Failed to load pull requests.</p>
        ) : isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading pull requests…</p>
        ) : pullRequests.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">No pull requests yet.</p>
        ) : (
          <ul>
            {pullRequests.map((pullRequest) => {
              const creator = pullRequestCreatorLabel(pullRequest.createdBy);
              return (
                <li
                  className="flex items-center gap-2 py-1.5"
                  key={pullRequest.id ?? `${pullRequest.url}-${pullRequest.number}`}
                >
                  <WorkOrderPullRequestInline
                    className="min-w-0 flex-1 justify-start"
                    pullRequest={pullRequest}
                    showTitle
                  />
                  {creator ? (
                    <span
                      className="shrink-0 truncate text-[12px] text-muted-foreground"
                      title={`Created by ${creator} on SuperPlane`}
                    >
                      by {creator}
                    </span>
                  ) : null}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}
