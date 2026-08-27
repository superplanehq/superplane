import type { FactoriesFactoryPullRequest } from "@/api-client";

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
            {pullRequests.map((pullRequest) => (
              <li
                className="flex items-center py-1.5"
                key={pullRequest.id ?? `${pullRequest.url}-${pullRequest.number}`}
              >
                <WorkOrderPullRequestInline className="w-full justify-start" pullRequest={pullRequest} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
