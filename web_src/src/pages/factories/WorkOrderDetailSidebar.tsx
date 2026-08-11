import type {
  FactoriesAutomationRef,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderExecution,
} from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { appPath } from "@/lib/appPaths";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { Forward, Loader2, Plus } from "lucide-react";
import { Link } from "react-router-dom";

import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import { factoryLineDetailPath } from "./lib/factoryPagePaths";
import { WorkOrderArtifactsPanel } from "./WorkOrderArtifactsPanel";
import { WorkOrderAssigneesField } from "./WorkOrderAssigneesField";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailSidebarProps {
  organizationId: string;
  order: FactoriesWorkOrder;
  artifacts: FactoriesWorkOrderArtifact[];
  isArtifactsLoading: boolean;
  artifactsError?: Error | null;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  factoryLines: FactoriesFactoryLine[];
  factoryId: string;
  canAssign: boolean;
  canDispatch: boolean;
  permissionsLoading: boolean;
  isAssigneesSaving: boolean;
  isDispatchable: boolean;
  isDispatching: boolean;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
}

/**
 * Right-hand column for the work order detail page: overview (status /
 * creator / assignee / date / cost), factory lines that ran (with dispatch),
 * and artifacts. Keeps the surrounding aside layout free of business logic.
 */
export function WorkOrderDetailSidebar({
  organizationId,
  order,
  artifacts,
  isArtifactsLoading,
  artifactsError,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  factoryLines,
  factoryId,
  canAssign,
  canDispatch,
  permissionsLoading,
  isAssigneesSaving,
  isDispatchable,
  isDispatching,
  onAssigneesSave,
  onDispatch,
}: WorkOrderDetailSidebarProps) {
  return (
    <div className="flex flex-col gap-6">
      <OverviewSection
        organizationId={organizationId}
        order={order}
        displayStatus={displayStatus}
        statusMeta={statusMeta}
        assigneeIds={assigneeIds}
        assigneeNames={assigneeNames}
        canAssign={canAssign}
        isAssigneesSaving={isAssigneesSaving}
        onAssigneesSave={onAssigneesSave}
      />

      <FactoryLinesSection
        organizationId={organizationId}
        factoryId={factoryId}
        executions={order.executions ?? []}
        factoryLines={factoryLines}
        canDispatch={canDispatch}
        permissionsLoading={permissionsLoading}
        isDispatchable={isDispatchable}
        isDispatching={isDispatching}
        onDispatch={onDispatch}
      />

      <WorkOrderArtifactsPanel artifacts={artifacts} isLoading={isArtifactsLoading} error={artifactsError} />
    </div>
  );
}

function OverviewSection({
  organizationId,
  order,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
}: {
  organizationId: string;
  order: FactoriesWorkOrder;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
}) {
  const createdAt = order.createdAt ? new Date(order.createdAt) : null;
  const totalTokens = parseNumericFromString(order.totalTokens);
  const totalCostCents = parseNumericFromString(order.totalCostCents);
  const showUsage = totalTokens > 0 || totalCostCents > 0;

  return (
    <section>
      <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Overview</h2>

      <dl className="mt-3 space-y-3">
        <OverviewRow label="Status">
          <Badge
            variant="outline"
            className={cn("inline-flex px-2 py-0.5 text-[11px] font-medium", statusMeta.className)}
          >
            {displayStatus === "running" ? <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden /> : null}
            {statusMeta.label}
          </Badge>
        </OverviewRow>

        <OverviewRow label="Created by">
          <CreatorValue organizationId={organizationId} order={order} />
        </OverviewRow>

        <OverviewRow label="Assignee">
          <div className="flex-1">
            <WorkOrderAssigneesField
              organizationId={organizationId}
              assigneeIds={assigneeIds}
              assigneeNames={assigneeNames}
              canEdit={canAssign}
              isSaving={isAssigneesSaving}
              onSave={onAssigneesSave}
            />
          </div>
        </OverviewRow>

        {createdAt ? (
          <OverviewRow label="Created">
            <span className="text-sm text-gray-700 dark:text-gray-300" title={createdAt.toLocaleString()}>
              {formatTimeAgo(createdAt)}
            </span>
          </OverviewRow>
        ) : null}

        {showUsage ? (
          <OverviewRow label="Usage">
            <span className="text-sm text-gray-700 dark:text-gray-300">
              {formatUsageLine(totalTokens, totalCostCents)}
            </span>
          </OverviewRow>
        ) : null}
      </dl>
    </section>
  );
}

function OverviewRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <dt className="w-24 shrink-0 text-xs text-gray-500 dark:text-gray-400">{label}</dt>
      <dd className="flex min-w-0 flex-1 items-center justify-end gap-2 text-right">{children}</dd>
    </div>
  );
}

function CreatorValue({ organizationId, order }: { organizationId: string; order: FactoriesWorkOrder }) {
  const automation = order.createdByAutomation;
  if (isAutomationRefResolved(automation)) {
    return <AutomationLink organizationId={organizationId} automation={automation} />;
  }

  const name = order.createdBy?.name ?? "Unknown";
  return <span className="truncate text-sm text-gray-700 dark:text-gray-300">{name}</span>;
}

function isAutomationRefResolved(ref: FactoriesAutomationRef | undefined): ref is FactoriesAutomationRef {
  return Boolean(ref && (ref.nodeName || ref.appName));
}

function AutomationLink({
  organizationId,
  automation,
}: {
  organizationId: string;
  automation: FactoriesAutomationRef;
}) {
  const label = automation.nodeName || automation.appName || "Automation";
  if (automation.appId) {
    return (
      <Link
        to={appPath(organizationId, automation.appId)}
        className="truncate text-sm text-violet-700 hover:underline dark:text-violet-300"
      >
        {label}
      </Link>
    );
  }
  return <span className="truncate text-sm text-gray-700 dark:text-gray-300">{label}</span>;
}

function FactoryLinesSection({
  organizationId,
  factoryId,
  executions,
  factoryLines,
  canDispatch,
  permissionsLoading,
  isDispatchable,
  isDispatching,
  onDispatch,
}: {
  organizationId: string;
  factoryId: string;
  executions: FactoriesWorkOrderExecution[];
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  permissionsLoading: boolean;
  isDispatchable: boolean;
  isDispatching: boolean;
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
}) {
  const rows = deriveLineRows(executions);

  return (
    <section>
      <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Factory lines</h2>

      <ul className="mt-3 space-y-1.5">
        {rows.length === 0 ? (
          <li className="text-sm text-gray-500 dark:text-gray-400">This work order has not run on a line yet.</li>
        ) : (
          rows.map((row) => (
            <LineRow key={row.lineId} organizationId={organizationId} factoryId={factoryId} row={row} />
          ))
        )}
      </ul>

      {isDispatchable ? (
        <div className="mt-2">
          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
            <DispatchWorkOrderPopover
              lines={factoryLines}
              isSaving={isDispatching}
              canDispatch={canDispatch || permissionsLoading}
              onDispatch={onDispatch}
              align="start"
            >
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={!canDispatch || factoryLines.length === 0}
                className="h-7 justify-start gap-1.5 px-2 text-xs text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
                data-testid="work-order-sidebar-dispatch-button"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden />
                Send to line
                <Forward className="ml-auto h-3.5 w-3.5" aria-hidden />
              </Button>
            </DispatchWorkOrderPopover>
          </PermissionTooltip>
        </div>
      ) : null}
    </section>
  );
}

interface FactoryLineRow {
  lineId: string;
  lineName: string;
  tone: "success" | "warning" | "danger" | "muted";
}

function deriveLineRows(executions: FactoriesWorkOrderExecution[]): FactoryLineRow[] {
  const map = new Map<string, FactoryLineRow>();
  for (const execution of executions) {
    const lineId = execution.line?.id ?? "unknown";
    const lineName = execution.line?.name?.trim() || "Unnamed line";
    const tone = executionTone(execution);
    const existing = map.get(lineId);
    if (!existing) {
      map.set(lineId, { lineId, lineName, tone });
      continue;
    }
    existing.tone = worstTone(existing.tone, tone);
  }
  return [...map.values()];
}

function executionTone(execution: FactoriesWorkOrderExecution): FactoryLineRow["tone"] {
  if (execution.result === "RESULT_FAILED") return "danger";
  if (
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING"
  ) {
    return "warning";
  }
  if (execution.result === "RESULT_PASSED") return "success";
  return "muted";
}

// Line rows collapse many executions into a single dot. "worst" surfaces
// failure over running over completed so operators see the highest-priority
// state at a glance.
function worstTone(a: FactoryLineRow["tone"], b: FactoryLineRow["tone"]): FactoryLineRow["tone"] {
  const priority: Record<FactoryLineRow["tone"], number> = {
    danger: 3,
    warning: 2,
    success: 1,
    muted: 0,
  };
  return priority[a] >= priority[b] ? a : b;
}

const TONE_DOT_CLASS: Record<FactoryLineRow["tone"], string> = {
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  muted: "bg-gray-300 dark:bg-gray-600",
};

function LineRow({
  organizationId,
  factoryId,
  row,
}: {
  organizationId: string;
  factoryId: string;
  row: FactoryLineRow;
}) {
  const isLinkable = row.lineId && row.lineId !== "unknown";
  const inner = (
    <>
      <span className={cn("h-2 w-2 rounded-full", TONE_DOT_CLASS[row.tone])} aria-hidden />
      <span className="min-w-0 truncate text-sm text-gray-800 dark:text-gray-200">{row.lineName}</span>
    </>
  );

  return (
    <li>
      {isLinkable ? (
        <Link
          to={factoryLineDetailPath(organizationId, factoryId, row.lineId)}
          className="inline-flex min-w-0 items-center gap-2 rounded-md px-2 py-1 hover:bg-gray-50 dark:hover:bg-gray-800/60"
        >
          {inner}
        </Link>
      ) : (
        <span className="inline-flex min-w-0 items-center gap-2 px-2 py-1">{inner}</span>
      )}
    </li>
  );
}

function parseNumericFromString(value: string | undefined): number {
  if (!value) return 0;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatUsageLine(totalTokens: number, totalCostCents: number): string {
  const parts: string[] = [];
  if (totalTokens > 0) parts.push(formatCompactTokens(totalTokens));
  if (totalCostCents > 0) parts.push(formatUsdCents(totalCostCents));
  return parts.join(" · ");
}

function formatCompactTokens(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M tokens`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(0)}k tokens`;
  return `${tokens} tokens`;
}

function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}
