import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { SidebarSectionHeading } from "../../sidebar/SidebarPrimitives";
import { WorkOrderArtifactsList } from "../../WorkOrderArtifactsList";
import { WorkOrderPullRequestsList } from "../../WorkOrderPullRequestsList";
import { splitRunLinkedArtifacts } from "./splitRunPopupModel";
import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

/**
 * Source, artifacts, and pull requests. Used on the left pane after Start.
 */
export function WorkOrderSplitRunOverviewSidebar({
  source,
  artifacts,
  artifactsLoading = false,
  pullRequests = [],
  pullRequestsLoading = false,
  pullRequestsError = null,
}: {
  source?: SplitRunSource;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading?: boolean;
  pullRequests?: FactoriesFactoryPullRequest[];
  pullRequestsLoading?: boolean;
  pullRequestsError?: Error | null;
}) {
  return (
    <aside className="h-full overflow-y-auto px-6 py-6" data-testid="split-run-overview-sidebar">
      <div className="flex flex-col gap-6">
        <section aria-label="Source">
          <SidebarSectionHeading>Source</SidebarSectionHeading>
          {source ? (
            <WorkOrderSplitRunSource source={source} />
          ) : (
            <p className="mt-2 text-[13px] text-muted-foreground">No source yet.</p>
          )}
        </section>
        <WorkOrderArtifactsList artifacts={splitRunLinkedArtifacts(artifacts, source)} isLoading={artifactsLoading} />
        <WorkOrderPullRequestsList
          pullRequests={pullRequests}
          isLoading={pullRequestsLoading}
          error={pullRequestsError}
        />
      </div>
    </aside>
  );
}
