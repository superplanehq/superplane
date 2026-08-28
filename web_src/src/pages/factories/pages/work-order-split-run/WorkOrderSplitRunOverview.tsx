import type { FactoriesWorkOrderArtifact } from "@/api-client";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { SidebarSectionHeading } from "../../sidebar/SidebarPrimitives";
import { WorkOrderArtifactsList } from "../../WorkOrderArtifactsList";
import { WorkOrderCheckComment } from "../../WorkOrderCheckComment";
import { SplitRunReview } from "./SplitRunReview";
import type { SplitRunFooter } from "./splitRunFooter";
import { SPLIT_RUN_PANE_GRID_CLASSNAME, splitRunLinkedArtifacts } from "./splitRunPopupModel";
import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunDescription } from "./WorkOrderSplitRunDescription";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

/**
 * Description tab: reading column on the left, Source and Artifacts on the
 * right.
 */
export function WorkOrderSplitRunOverview({
  description,
  artifacts,
  artifactsLoading = false,
  checks,
  organizationId,
  factoryKey,
  orderNumber,
  expandFirstCheck = false,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  footer,
}: {
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading?: boolean;
  checks: WorkOrderCheckPresentation[];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  expandFirstCheck?: boolean;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  source?: SplitRunSource;
  footer?: SplitRunFooter;
}) {
  return (
    <div className={SPLIT_RUN_PANE_GRID_CLASSNAME} data-testid="split-run-work-order-tab">
      <div className="flex min-h-0 flex-col border-b border-border md:border-r md:border-b-0">
        <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
          <WorkOrderSplitRunDescription
            description={description}
            canEdit={canEditDescription}
            busy={descriptionBusy}
            onSave={onDescriptionSave}
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
        {footer ? (
          <SplitRunReview
            footer={footer}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
          />
        ) : null}
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
          <WorkOrderArtifactsList artifacts={splitRunLinkedArtifacts(artifacts, source)} isLoading={artifactsLoading} />
        </div>
      </aside>
    </div>
  );
}
