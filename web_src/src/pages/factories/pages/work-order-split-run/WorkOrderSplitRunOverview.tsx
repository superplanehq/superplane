import type { FactoriesWorkOrderArtifact } from "@/api-client";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { SidebarSectionHeading } from "../../sidebar/SidebarPrimitives";
import { WorkOrderArtifactsList } from "../../WorkOrderArtifactsList";
import { WorkOrderCheckComment } from "../../WorkOrderCheckComment";
import { splitRunLinkedArtifacts } from "./splitRunPopupModel";
import { WorkOrderSplitRunDescription } from "./WorkOrderSplitRunDescription";

/**
 * Description tab: reading column on the left, Source and Artifacts on the
 * right — the same sidebar model as the work-order page.
 */
export function WorkOrderSplitRunOverview({
  description,
  artifacts,
  checks,
  organizationId,
  factoryKey,
  orderNumber,
  expandFirstCheck = false,
  canEditDescription = false,
}: {
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  checks: WorkOrderCheckPresentation[];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  expandFirstCheck?: boolean;
  canEditDescription?: boolean;
}) {
  return (
    <div
      className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_minmax(16rem,20rem)] gap-x-[var(--workspace-column-gap)]"
      data-testid="split-run-work-order-tab"
    >
      <div className="min-h-0 overflow-y-auto px-8 py-6">
        <WorkOrderSplitRunDescription description={description} canEdit={canEditDescription} />

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

      <aside className="min-h-0 overflow-y-auto py-6 pr-6" data-testid="split-run-overview-sidebar">
        <div className="flex flex-col gap-6">
          <section aria-label="Source">
            <SidebarSectionHeading>Source</SidebarSectionHeading>
            <p className="mt-2 text-[13px] text-muted-foreground">No source yet.</p>
          </section>
          <WorkOrderArtifactsList artifacts={splitRunLinkedArtifacts(artifacts)} isLoading={false} />
        </div>
      </aside>
    </div>
  );
}
