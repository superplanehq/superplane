import type { FactoriesFactoryLine } from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Link } from "react-router";
import { workOrderOpenPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { formatCompactTokens, formatUsdCents } from "../lib/workOrderUsage";
import { WorkOrderLineStep } from "./WorkOrderLineStep";
import { AssigneeGroup, InlineDispatchButton } from "./WorkOrderRowActions";

interface WorkOrdersTableViewProps {
  entries: WorkOrderListEntry[];
  organizationId: string;
  factoryKey: string;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canAssign: boolean;
  /** Tasks with a dispatch in flight. Only their controls show a busy state. */
  dispatchingOrderIds: ReadonlySet<string>;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Shared grid-template-columns tracks for the header and every row. Every
 * track is a fixed length or `1fr` — never `auto` — so the header and each
 * row (which are separate CSS grid formatting contexts) always resolve to
 * identical pixel widths and stay aligned regardless of row content (e.g.
 * whether a row has an avatar or a dispatch button in the Owner column).
 * Keep the header `<div>` and row `<article>` className referencing this
 * constant so the two templates can never drift apart.
 */
const TABLE_GRID_COLS =
  "grid-cols-[118px_52px_1fr_88px_96px] md:grid-cols-[130px_60px_1fr_120px_96px_100px_110px] lg:grid-cols-[130px_70px_1fr_160px_110px_110px_120px]";

/**
 * Responsive table. Columns collapse gracefully on narrower viewports —
 * Updated and Line hide below md. Status, ID, Title, Spend, and Owner stay
 * visible in every layout so token and USD cost stay visible on the main
 * work-order list.
 */
export function WorkOrdersTableView(props: WorkOrdersTableViewProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card" data-testid="work-orders-table">
      <div
        className={cn(
          "grid items-center gap-3 border-b border-border bg-muted/30 px-3 py-2 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground",
          TABLE_GRID_COLS,
        )}
      >
        <span>Status</span>
        <span>ID</span>
        <span>Title</span>
        <span className="hidden md:inline">Line</span>
        <span className="text-right">Spend</span>
        <span className="hidden md:inline">Updated</span>
        <span className="text-right">Owner</span>
      </div>
      <ul className="divide-y divide-border">
        {props.entries.map((entry) => (
          <li key={entry.id}>
            <TableRow entry={entry} {...props} />
          </li>
        ))}
      </ul>
    </div>
  );
}

function TableRow({
  entry,
  organizationId,
  factoryKey,
  factoryLines,
  canDispatch,
  canAssign,
  dispatchingOrderIds,
  isAssigneesSaving,
  onDispatch,
  onAssigneesSave,
}: WorkOrdersTableViewProps & { entry: WorkOrderListEntry }) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const href = workOrderOpenPath(organizationId, factoryKey, entry.order.number, factoryLines[0]?.id);
  const timeLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";
  return (
    <article
      className={cn(
        "group relative grid items-center gap-3 px-3 py-2 transition-colors hover:bg-accent/40",
        TABLE_GRID_COLS,
      )}
      data-testid={`work-orders-table-row-${entry.id}`}
    >
      <Link to={href} className="absolute inset-0 z-0" aria-label={`Open ${entry.title}`} />

      <span
        className={cn(
          "relative z-10 pointer-events-none inline-flex w-fit items-center gap-1 rounded-full border px-1.5 py-0.5 text-[10px] font-medium",
          meta.className,
        )}
      >
        <span className={cn("size-1.5 rounded-full", meta.dotClassName)} aria-hidden />
        {meta.label}
      </span>

      <span className="relative z-10 pointer-events-none truncate font-mono text-[11px] text-muted-foreground">
        {entry.displayKey}
      </span>

      <p className="relative z-10 pointer-events-none min-w-0 truncate text-[13px] font-medium text-foreground">
        {entry.title}
      </p>

      <WorkOrderLineStep
        entry={entry}
        className="relative z-10 pointer-events-none hidden min-w-0 md:inline-flex"
        fallback="—"
      />

      <SpendCell entry={entry} />

      <span className="relative z-10 pointer-events-none hidden text-[11px] text-muted-foreground md:inline">
        {timeLabel}
      </span>

      <div className="relative z-10 pointer-events-none flex items-center justify-end gap-1">
        <InlineDispatchButton
          entry={entry}
          lines={factoryLines}
          canDispatch={canDispatch}
          isDispatching={dispatchingOrderIds.has(entry.id)}
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

function SpendCell({ entry }: { entry: WorkOrderListEntry }) {
  const usd = entry.totalCostCents > 0 ? formatUsdCents(entry.totalCostCents) : null;
  const tokens = entry.totalTokens > 0 ? formatCompactTokens(entry.totalTokens) : null;
  if (!usd && !tokens) {
    return (
      <span className="relative z-10 pointer-events-none text-right text-[11px] tabular-nums text-muted-foreground">
        —
      </span>
    );
  }

  return (
    <span
      className="relative z-10 pointer-events-none text-right text-[11px] leading-tight tabular-nums text-muted-foreground"
      title={entry.usageTooltip ?? undefined}
    >
      {usd ? <span className="block">{usd}</span> : null}
      {tokens ? <span className="block">{tokens}</span> : null}
    </span>
  );
}
