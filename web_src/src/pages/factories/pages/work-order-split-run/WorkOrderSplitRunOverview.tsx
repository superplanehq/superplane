import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { MarkdownContent } from "@/pages/app/Markdown";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { WorkOrderArtifactsList } from "../../WorkOrderArtifactsList";
import { WorkOrderCheckCard } from "../../WorkOrderChecksSection";

/**
 * Description tab: markdown on the left, checks and artifacts on the right.
 */
export function WorkOrderSplitRunOverview({
  description,
  artifacts,
  checks,
  organizationId,
  factoryKey,
  orderNumber,
}: {
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  checks: WorkOrderCheckPresentation[];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
}) {
  return (
    <div
      className="grid min-h-0 flex-1 grid-cols-[minmax(0,3fr)_minmax(16rem,2fr)]"
      data-testid="split-run-work-order-tab"
    >
      <section className="min-h-0 overflow-y-auto px-5 py-4" aria-label="Description">
        {description.trim() ? (
          <MarkdownContent content={description} variant="workspace" data-testid="split-run-description" />
        ) : (
          <p className="text-[13px] text-muted-foreground">No description yet.</p>
        )}
      </section>

      <aside
        className="min-h-0 overflow-y-auto border-l border-border px-5 py-4"
        data-testid="split-run-overview-sidebar"
      >
        {checks.length > 0 ? (
          <section className="mb-8" data-testid="split-run-overview-checks">
            <h3 className="workspace-section-label">Checks</h3>
            <ul className="mt-3 flex flex-col gap-3">
              {checks.map((check) => (
                <li key={check.id}>
                  <WorkOrderCheckCard
                    check={check}
                    runHref={
                      organizationId && factoryKey
                        ? getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })
                        : null
                    }
                  />
                </li>
              ))}
            </ul>
          </section>
        ) : null}
        <WorkOrderArtifactsList artifacts={artifacts} isLoading={false} />
      </aside>
    </div>
  );
}
