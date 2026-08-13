import type { FactoriesFactoryLine } from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Link } from "react-router";
import { workOrderDetailPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { WorkOrderLineStep } from "./WorkOrderLineStep";
import { AssigneeGroup, InlineDispatchButton } from "./WorkOrderRowActions";

interface WorkOrdersTableViewProps {
  entries: WorkOrderListEntry[];
  organizationId: string;
  factoryId: string;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canAssign: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Responsive table. Columns collapse gracefully on narrower viewports —
 * Spend and Updated hide first, Line hides second. Status/ID/Title stay
 * visible in every layout.
 */
export function WorkOrdersTableView(props: WorkOrdersTableViewProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card" data-testid="work-orders-table">
      <div className="grid grid-cols-[110px_60px_1fr_auto] items-center gap-3 border-b border-border bg-muted/30 px-3 py-2 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground md:grid-cols-[110px_60px_1fr_140px_120px_auto] lg:grid-cols-[110px_70px_1fr_180px_100px_120px_auto]">
        <span>Status</span>
        <span>ID</span>
        <span>Title</span>
        <span className="hidden md:inline">Line</span>
        <span className="hidden lg:inline">Spend</span>
        <span className="hidden md:inline">Updated</span>
        <span className="text-right">Assignee</span>
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
  factoryId,
  factoryLines,
  canDispatch,
  canAssign,
  isDispatching,
  isAssigneesSaving,
  onDispatch,
  onAssigneesSave,
}: WorkOrdersTableViewProps & { entry: WorkOrderListEntry }) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const href = workOrderDetailPath(organizationId, factoryId, entry.id);
  const timeLabel = entry.updatedAtMs > 0 ? formatTimeAgo(new Date(entry.updatedAtMs)) : "—";
  return (
    <article
      className="group relative grid grid-cols-[110px_60px_1fr_auto] items-center gap-3 px-3 py-2 transition-colors hover:bg-accent/40 md:grid-cols-[110px_60px_1fr_140px_120px_auto] lg:grid-cols-[110px_70px_1fr_180px_100px_120px_auto]"
      data-testid={`work-orders-table-row-${entry.id}`}
    >
      <Link to={href} className="absolute inset-0 z-0" aria-label={`Open ${entry.title}`} />

      <span
        className={cn(
          "relative z-10 pointer-events-none inline-flex w-fit items-center gap-1 rounded-full border px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.04em]",
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

      <span
        className="relative z-10 pointer-events-none hidden text-[11px] text-muted-foreground lg:inline"
        title={entry.usageTooltip ?? undefined}
      >
        {entry.usageLabel ?? "—"}
      </span>

      <span className="relative z-10 pointer-events-none hidden text-[11px] text-muted-foreground md:inline">
        {timeLabel}
      </span>

      <div className="relative z-10 pointer-events-none flex items-center justify-end gap-1">
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
