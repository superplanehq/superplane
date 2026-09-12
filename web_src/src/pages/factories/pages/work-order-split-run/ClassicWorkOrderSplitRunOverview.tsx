import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { SidebarSectionHeading } from "../../sidebar/SidebarPrimitives";
import { WorkOrderArtifactsList } from "../../WorkOrderArtifactsList";
import { WorkOrderCheckComment } from "../../WorkOrderCheckComment";
import { WorkOrderPullRequestsList } from "../../WorkOrderPullRequestsList";
import { classicSplitRunLinkedArtifacts, SPLIT_RUN_PANE_GRID_CLASSNAME } from "./splitRunPopupModel";
import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunDescription } from "./WorkOrderSplitRunDescription";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

type ClassicWorkOrderSplitRunOverviewProps = {
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading?: boolean;
  pullRequests?: FactoriesFactoryPullRequest[];
  pullRequestsLoading?: boolean;
  pullRequestsError?: Error | null;
  checks: WorkOrderCheckPresentation[];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  expandFirstCheck?: boolean;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  source?: SplitRunSource;
  files?: FilesFile[];
  sessionLookupError?: string;
};

/** The task details shown when live refinement is not active. */
export function ClassicWorkOrderSplitRunOverview({
  description,
  artifacts,
  artifactsLoading = false,
  pullRequests = [],
  pullRequestsLoading = false,
  pullRequestsError = null,
  checks,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  expandFirstCheck = false,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  files,
  sessionLookupError,
}: ClassicWorkOrderSplitRunOverviewProps) {
  return (
    <div className={SPLIT_RUN_PANE_GRID_CLASSNAME} data-testid="split-run-work-order-tab">
      <div className="flex min-h-0 flex-col border-b border-border md:border-r md:border-b-0">
        <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
          {sessionLookupError ? (
            <p className="mb-4 text-[13px] text-destructive" role="alert">
              The refinement session did not load. Refresh the page to try again.
            </p>
          ) : null}
          <WorkOrderSplitRunDescription
            description={description}
            canEdit={canEditDescription}
            busy={descriptionBusy}
            onSave={onDescriptionSave}
            files={files}
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
          />

          {checks.length > 0 ? (
            <section className="mt-10" data-testid="split-run-overview-checks" aria-label="Checks">
              <h3 className="workspace-section-label">Checks</h3>
              <div className="mt-1">
                {checks.map((check, index) => (
                  <WorkOrderCheckComment
                    key={check.id}
                    check={check}
                    defaultOpen={expandFirstCheck && index === 0}
                    runHref={
                      organizationId && factoryKey
                        ? getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })
                        : null
                    }
                  />
                ))}
              </div>
            </section>
          ) : null}
        </div>
      </div>

      <aside className="min-h-0 overflow-y-auto px-6 py-6" data-testid="split-run-overview-sidebar">
        <div className="flex flex-col gap-6">
          <section aria-label="Source">
            <SidebarSectionHeading>Source</SidebarSectionHeading>
            {source ? (
              <WorkOrderSplitRunSource source={source} />
            ) : (
              <p className="mt-2 text-[13px] text-muted-foreground">No source yet.</p>
            )}
          </section>
          <WorkOrderArtifactsList
            artifacts={classicSplitRunLinkedArtifacts(artifacts, source)}
            isLoading={artifactsLoading}
          />
          <WorkOrderPullRequestsList
            pullRequests={pullRequests}
            isLoading={pullRequestsLoading}
            error={pullRequestsError}
          />
        </div>
      </aside>
    </div>
  );
}
