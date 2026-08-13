import type { FactoriesFactoryLine } from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Link } from "react-router";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
import { WORK_ORDER_BOARD_LANES, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { WorkOrderLineStep } from "./WorkOrderLineStep";
import { AssigneeGroup, InlineDispatchButton } from "./WorkOrderRowActions";

interface WorkOrdersListViewProps {
  entries: WorkOrderListEntry[];
  organizationId: string;
  factoryKey: string;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canAssign: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Dense grouped list. Uses the same four board lanes as section headers so
 * users have a single mental model across layouts.
 */
export function WorkOrdersListView(props: WorkOrdersListViewProps) {
  const grouped = groupWorkOrderEntriesByLane(props.entries);
  return (
    <div className="flex flex-col gap-4" data-testid="work-orders-list">
      {WORK_ORDER_BOARD_LANES.map((lane) => {
        const laneEntries = grouped.get(lane.id) ?? [];
        if (laneEntries.length === 0) {
          return null;
        }
        return (
          <section key={lane.id} aria-label={lane.title} data-testid={`work-orders-list-lane-${lane.id}`}>
            <header className="mb-1.5 flex items-center gap-2 px-1">
              <h2 className="text-[12px] font-semibold uppercase tracking-[0.06em] text-foreground/80">{lane.title}</h2>
              <span className="rounded-full bg-muted px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                {laneEntries.length}
              </span>
            </header>
            <ul className="divide-y divide-border overflow-hidden rounded-md border border-border bg-card">
              {laneEntries.map((entry) => (
                <li key={entry.id}>
                  <ListRow entry={entry} {...props} />
                </li>
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}

function ListRow({
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
}: WorkOrdersListViewProps & { entry: WorkOrderListEntry }) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const href =
    entry.order.number !== undefined ? workOrderDetailPath(organizationId, factoryKey, entry.order.number) : "#";
  const timeLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";
  return (
    <article
      className="group relative flex items-center gap-3 px-3 py-2 transition-colors hover:bg-accent/40"
      data-testid={`work-orders-list-row-${entry.id}`}
    >
      <Link to={href} className="absolute inset-0 z-0" aria-label={`Open ${entry.title}`} />

      <span
        className={cn(
          "relative z-10 inline-flex shrink-0 items-center gap-1 rounded-full border px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.04em]",
          meta.className,
        )}
      >
        <span className={cn("size-1.5 rounded-full", meta.dotClassName)} aria-hidden />
        {meta.label}
      </span>

      <span className="relative z-10 shrink-0 font-mono text-[11px] text-muted-foreground w-[54px]">
        {entry.displayKey}
      </span>

      <div className="relative z-10 min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{entry.title}</p>
        <WorkOrderLineStep entry={entry} className="mt-0.5 max-w-full" />
      </div>

      {entry.usageLabel ? (
        <span
          className="relative z-10 hidden text-[11px] text-muted-foreground sm:inline"
          title={entry.usageTooltip ?? undefined}
        >
          {entry.usageLabel}
        </span>
      ) : null}

      <span className="relative z-10 hidden shrink-0 text-[11px] text-muted-foreground sm:inline">{timeLabel}</span>

      <div className="relative z-10 flex shrink-0 items-center gap-1">
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
    </article>
  );
}
