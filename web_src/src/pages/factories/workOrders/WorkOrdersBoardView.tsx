import type { FactoriesFactoryLine } from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Link } from "react-router";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
import {
  WORK_ORDER_BOARD_LANES,
  getWorkOrderDisplayStatusMeta,
  type WorkOrderBoardLaneDefinition,
} from "../lib/workOrderProgress";
import { WorkOrderLineStep } from "./WorkOrderLineStep";
import { AssigneeGroup, InlineDispatchButton } from "./WorkOrderRowActions";

interface WorkOrdersBoardViewProps {
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

/** Four-lane Kanban-style board mapping to the shared display statuses. */
export function WorkOrdersBoardView(props: WorkOrdersBoardViewProps) {
  const grouped = groupWorkOrderEntriesByLane(props.entries);
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4" data-testid="work-orders-board">
      {WORK_ORDER_BOARD_LANES.map((lane) => (
        <BoardLane key={lane.id} {...props} lane={lane} entries={grouped.get(lane.id) ?? []} />
      ))}
    </div>
  );
}

interface BoardLaneProps extends WorkOrdersBoardViewProps {
  lane: WorkOrderBoardLaneDefinition;
  entries: WorkOrderListEntry[];
}

function BoardLane({ lane, entries, ...rest }: BoardLaneProps) {
  const laneBg = laneBackground(lane.id);
  return (
    <section
      aria-label={lane.title}
      className={cn("flex min-h-[160px] flex-col rounded-lg border border-border/70 p-2", laneBg)}
      data-testid={`work-orders-board-lane-${lane.id}`}
    >
      <header className="flex items-center justify-between px-2 pb-2">
        <div className="inline-flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-[0.06em] text-foreground/80">{lane.title}</h2>
          <span className="rounded-full bg-background/70 px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
            {entries.length}
          </span>
        </div>
      </header>

      {entries.length === 0 ? (
        <p className="mt-2 rounded-md border border-dashed border-border/60 px-3 py-6 text-center text-[12px] text-muted-foreground">
          {lane.description}
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {entries.map((entry) => (
            <li key={entry.id}>
              <BoardCard entry={entry} {...rest} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function laneBackground(laneId: string): string {
  switch (laneId) {
    case "running":
      return "bg-[color:var(--status-running-lane-bg)]";
    case "done":
      return "bg-[color:var(--status-done-lane-bg)]";
    default:
      return "bg-card/60";
  }
}

type BoardCardProps = Omit<WorkOrdersBoardViewProps, "entries"> & { entry: WorkOrderListEntry };

function BoardCard({
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
}: BoardCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const href =
    entry.order.number !== undefined ? workOrderDetailPath(organizationId, factoryKey, entry.order.number) : "#";
  const timeLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";
  return (
    <article className="group relative rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow">
      <Link to={href} className="absolute inset-0 z-0 rounded-md" aria-label={`Open ${entry.title}`} />

      <div className="relative z-10 flex items-start justify-between gap-2">
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

      <h3 className="relative z-10 mt-1.5 line-clamp-2 text-[13px] font-medium leading-snug text-foreground">
        {entry.title}
      </h3>

      <WorkOrderLineStep entry={entry} className="relative z-10 mt-1 max-w-full" />

      <div className="relative z-10 mt-2 flex items-center justify-between gap-2">
        <span className="text-[11px] text-muted-foreground">{timeLabel}</span>
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
    </article>
  );
}
