import type { FactoriesFactoryLine } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { Link } from "react-router";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { WorkOrderLineStep } from "./WorkOrderLineStep";
import { AssigneeGroup, InlineDispatchButton, type WorkOrderRowCallbacks } from "./WorkOrderRowActions";

export interface WorkOrderCardContext extends WorkOrderRowCallbacks {
  organizationId: string;
  factoryKey: string;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canAssign: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
}

export interface WorkOrderCardProps extends WorkOrderCardContext {
  entry: WorkOrderListEntry;
  /**
   * Overlay destination. Defaults to the work order. The Lines board
   * passes the canvas run so a click opens the same card in the run view.
   */
  href?: string;
}

/**
 * The canonical work order card.
 *
 * Every board uses this complete component. It owns its content, navigation,
 * dispatch action, and assignee avatars so callers cannot create card variants.
 */
export function WorkOrderCard({
  entry,
  organizationId,
  factoryKey,
  factoryLines,
  canDispatch,
  canAssign,
  isDispatching,
  isAssigneesSaving,
  onDispatch,
  onAssigneesSave,
  href,
}: WorkOrderCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const destination =
    href ??
    (entry.order.number !== undefined ? workOrderDetailPath(organizationId, factoryKey, entry.order.number) : "#");
  const updatedLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";

  return (
    <article
      className="group relative w-full rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`work-order-card-${entry.id}`}
    >
      <Link to={destination} className="absolute inset-0 z-0 rounded-md" aria-label={`Open ${entry.title}`} />

      <div className="relative z-10 pointer-events-none">
        <div className="flex items-start justify-between gap-2">
          <span
            className={cn(
              "inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.04em]",
              meta.className,
            )}
          >
            <span className={cn("size-1.5 rounded-full", meta.dotClassName)} aria-hidden />
            {meta.label}
          </span>
          <span className="font-mono text-[11px] text-muted-foreground">{entry.displayKey}</span>
        </div>

        <h3 className="mt-1.5 line-clamp-2 text-[13px] font-medium leading-snug text-foreground">{entry.title}</h3>
        <WorkOrderLineStep entry={entry} className="mt-1 max-w-full" />

        <div className="mt-2 flex items-center justify-between gap-2">
          <span className="text-[11px] text-muted-foreground">{updatedLabel}</span>
          <div className="flex items-center gap-1">
            {entry.usageLabel ? (
              <span className="text-[11px] text-muted-foreground" title={entry.usageTooltip ?? undefined}>
                {entry.usageLabel}
              </span>
            ) : null}
            <InlineDispatchButton
              entry={entry}
              lines={factoryLines}
              canDispatch={canDispatch}
              isDispatching={isDispatching}
              onDispatch={onDispatch}
              visible={entry.isDispatchable}
            />
            <AssigneeGroup
              entry={entry}
              organizationId={organizationId}
              canAssign={canAssign}
              isAssigneesSaving={isAssigneesSaving}
              onAssigneesSave={onAssigneesSave}
            />
          </div>
        </div>
      </div>
    </article>
  );
}
