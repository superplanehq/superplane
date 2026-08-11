import type {
  FactoriesAutomationRef,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderExecution,
} from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import {
  Calendar,
  CircleDollarSign,
  CircleDot,
  ChevronDown,
  Loader2,
  Plus,
  Sparkles,
  User,
  UserPlus,
} from "lucide-react";
import { Link } from "react-router-dom";

import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import { OrgUserReference } from "./OrgUserReference";
import { WorkOrderArtifactsList } from "./WorkOrderArtifactsList";
import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";
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
 * Right-hand column for the work order detail page. Renders three sections in
 * the mockup style: a single Overview list (status, creator, assignee(s),
 * created, spending), Factory Lines (only lines that have runs + compact
 * "Send to line" trigger), and a bare Artifacts list. Every row shares the
 * same icon+value shape so the panel reads as a compact metadata list.
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
        executions={order.executions ?? []}
        factoryLines={factoryLines}
        canDispatch={canDispatch}
        permissionsLoading={permissionsLoading}
        isDispatchable={isDispatchable}
        isDispatching={isDispatching}
        onDispatch={onDispatch}
      />

      <WorkOrderArtifactsList artifacts={artifacts} isLoading={isArtifactsLoading} error={artifactsError} />
    </div>
  );
}

function SidebarSectionHeading({ children }: { children: React.ReactNode }) {
  return <h3 className="workspace-section-label">{children}</h3>;
}

function OverviewRow({
  icon,
  srLabel,
  children,
}: {
  icon: React.ReactNode;
  srLabel: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-2 py-1.5">
      <span className="shrink-0 text-muted-foreground" aria-hidden>
        {icon}
      </span>
      <div className="min-w-0 flex-1 text-[13px] tracking-[-0.01em] text-foreground">
        <span className="sr-only">{srLabel}</span>
        {children}
      </div>
    </div>
  );
}

const STATUS_TEXT_CLASSNAME: Partial<Record<WorkOrderDisplayStatus, string>> = {
  completed: "text-[color:var(--status-success)]",
  running: "text-[color:var(--status-running)]",
  failed: "text-[color:var(--status-danger)]",
  closedFailed: "text-[color:var(--status-danger)]",
  rejected: "text-muted-foreground",
  open: "text-sky-700 dark:text-sky-300",
  draft: "text-muted-foreground",
};

const STATUS_DOT_CLASSNAME: Partial<Record<WorkOrderDisplayStatus, string>> = {
  completed: "bg-[var(--status-success-dot)]",
  running: "bg-[var(--status-running-dot)]",
  failed: "bg-[var(--status-danger-dot)]",
  closedFailed: "bg-[var(--status-danger-dot)]",
  rejected: "bg-gray-400",
  open: "bg-sky-500",
  draft: "bg-gray-400",
};

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
  const showSpending = totalTokens > 0 || totalCostCents > 0;

  return (
    <section>
      <SidebarSectionHeading>Overview</SidebarSectionHeading>
      <div className="mt-2">
        <OverviewRow icon={<CircleDot className="size-3.5" aria-hidden />} srLabel="Status">
          <StatusValue displayStatus={displayStatus} label={statusMeta.label} />
        </OverviewRow>

        <OverviewRow icon={<UserPlus className="size-3.5" aria-hidden />} srLabel="Author">
          <CreatorValue organizationId={organizationId} order={order} />
        </OverviewRow>

        <AssigneeOverviewRow
          organizationId={organizationId}
          assigneeIds={assigneeIds}
          assigneeNames={assigneeNames}
          canEdit={canAssign}
          isSaving={isAssigneesSaving}
          onSave={onAssigneesSave}
        />

        {createdAt ? (
          <OverviewRow icon={<Calendar className="size-3.5" aria-hidden />} srLabel="Created">
            <span title={createdAt.toLocaleString()}>{formatCompactDate(createdAt)}</span>
          </OverviewRow>
        ) : null}

        {showSpending ? (
          <OverviewRow icon={<CircleDollarSign className="size-3.5" aria-hidden />} srLabel="Spending">
            <span title={formatSpendingTooltip(totalTokens, totalCostCents)}>
              {formatSpendingLine(totalTokens, totalCostCents)}
            </span>
          </OverviewRow>
        ) : null}
      </div>
    </section>
  );
}

function StatusValue({ displayStatus, label }: { displayStatus: WorkOrderDisplayStatus; label: string }) {
  const textClass = STATUS_TEXT_CLASSNAME[displayStatus] ?? "text-foreground";
  const dotClass = STATUS_DOT_CLASSNAME[displayStatus] ?? "bg-muted-foreground";
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-[13px]", textClass)}>
      {displayStatus === "running" ? (
        <Loader2 className="size-3 animate-spin" aria-hidden />
      ) : (
        <span className={cn("size-1.5 shrink-0 rounded-full", dotClass)} aria-hidden />
      )}
      {label}
    </span>
  );
}

function CreatorValue({ organizationId, order }: { organizationId: string; order: FactoriesWorkOrder }) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const automation = order.createdByAutomation;
  if (isAutomationRefResolved(automation)) {
    return <AutomationLink organizationId={organizationId} automation={automation} />;
  }
  const display = resolveUser(order.createdBy?.id, order.createdBy?.name);
  if (display) {
    return <OrgUserReference display={display} size="xs" nameClassName="truncate text-[13px]" />;
  }
  return <span className="truncate">{order.createdBy?.name ?? "Unknown"}</span>;
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
        className="truncate text-foreground underline underline-offset-2 hover:no-underline"
      >
        {label}
      </Link>
    );
  }
  return <span className="truncate">{label}</span>;
}

function AssigneeOverviewRow({
  organizationId,
  assigneeIds,
  assigneeNames,
  canEdit,
  isSaving,
  onSave,
}: {
  organizationId: string;
  assigneeIds: string[];
  assigneeNames: string[];
  canEdit: boolean;
  isSaving: boolean;
  onSave: (assigneeIds: string[]) => Promise<void>;
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  return (
    <OverviewRow icon={<User className="size-3.5" aria-hidden />} srLabel="Assignee">
      <PermissionTooltip allowed={canEdit} message="You don't have permission to update assignees.">
        <WorkOrderAssigneesPopover
          organizationId={organizationId}
          selectedIds={assigneeIds}
          onSave={onSave}
          isSaving={isSaving}
          canEdit={canEdit}
          align="end"
        >
          <button
            type="button"
            disabled={!canEdit || isSaving}
            aria-label={
              assigneeIds.length > 0 ? `Assignee: ${assigneeNames.filter(Boolean).join(", ")}` : "Assign work order"
            }
            className="-my-1.5 -mr-1.5 flex w-full min-w-0 items-center gap-1.5 rounded-md py-1.5 pr-1.5 text-left text-[13px] tracking-[-0.01em] text-foreground outline-none transition-colors hover:bg-accent/60 focus-visible:bg-accent/60 disabled:cursor-not-allowed disabled:opacity-60"
            data-testid="work-order-edit-assignees"
          >
            <AssigneeButtonBody assigneeIds={assigneeIds} assigneeNames={assigneeNames} resolveUser={resolveUser} />
            <ChevronDown className="ml-auto size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </button>
        </WorkOrderAssigneesPopover>
      </PermissionTooltip>
    </OverviewRow>
  );
}

function AssigneeButtonBody({
  assigneeIds,
  assigneeNames,
  resolveUser,
}: {
  assigneeIds: string[];
  assigneeNames: string[];
  resolveUser: ReturnType<typeof useOrgUserLookup>["resolveUser"];
}) {
  if (assigneeIds.length === 0) {
    return <span className="min-w-0 truncate text-muted-foreground">Assign…</span>;
  }
  if (assigneeIds.length === 1) {
    const display = resolveUser(assigneeIds[0], assigneeNames[0]);
    return <OrgUserReference display={display} size="xs" nameClassName="truncate text-[13px]" />;
  }
  const first = resolveUser(assigneeIds[0], assigneeNames[0]);
  return (
    <>
      <OrgUserReference display={first} size="xs" nameClassName="truncate text-[13px]" />
      <span className="shrink-0 text-muted-foreground">+{assigneeIds.length - 1}</span>
    </>
  );
}

function FactoryLinesSection({
  executions,
  factoryLines,
  canDispatch,
  permissionsLoading,
  isDispatchable,
  isDispatching,
  onDispatch,
}: {
  executions: FactoriesWorkOrderExecution[];
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  permissionsLoading: boolean;
  isDispatchable: boolean;
  isDispatching: boolean;
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
}) {
  const rows = deriveLineRows(executions);
  const isDispatchDisabled = !canDispatch || factoryLines.length === 0;

  return (
    <section>
      <SidebarSectionHeading>Factory Lines</SidebarSectionHeading>

      <div className="mt-2">
        {rows.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">Not run on a line yet.</p>
        ) : (
          rows.map((row) => <FactoryLineRow key={row.lineId} row={row} />)
        )}

        {isDispatchable ? (
          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
            <DispatchWorkOrderPopover
              lines={factoryLines}
              isSaving={isDispatching}
              canDispatch={canDispatch || permissionsLoading}
              onDispatch={onDispatch}
              align="start"
            >
              <button
                type="button"
                disabled={isDispatchDisabled}
                aria-label="Send to line"
                className={cn(
                  "-mx-1.5 mt-0.5 flex w-[calc(100%+0.75rem)] items-center gap-2 rounded-md px-1.5 py-1.5 text-left text-[13px] tracking-[-0.01em] outline-none transition-colors",
                  "text-muted-foreground hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60",
                  "disabled:cursor-not-allowed disabled:opacity-60",
                )}
                data-testid="work-order-sidebar-dispatch-button"
              >
                <Plus className="size-3.5 shrink-0" aria-hidden />
                Send to line
                <ChevronDown className="ml-auto size-3.5 shrink-0 text-muted-foreground" aria-hidden />
              </button>
            </DispatchWorkOrderPopover>
          </PermissionTooltip>
        ) : null}
      </div>
    </section>
  );
}

interface FactoryLineRowModel {
  lineId: string;
  lineName: string;
  tone: "success" | "warning" | "danger" | "muted";
}

function deriveLineRows(executions: FactoriesWorkOrderExecution[]): FactoryLineRowModel[] {
  const map = new Map<string, FactoryLineRowModel>();
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

function executionTone(execution: FactoriesWorkOrderExecution): FactoryLineRowModel["tone"] {
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
function worstTone(a: FactoryLineRowModel["tone"], b: FactoryLineRowModel["tone"]): FactoryLineRowModel["tone"] {
  const priority: Record<FactoryLineRowModel["tone"], number> = {
    danger: 3,
    warning: 2,
    success: 1,
    muted: 0,
  };
  return priority[a] >= priority[b] ? a : b;
}

const TONE_DOT_CLASS: Record<FactoryLineRowModel["tone"], string> = {
  success: "bg-[var(--status-success-dot)]",
  warning: "bg-[var(--status-warning-dot)]",
  danger: "bg-[var(--status-danger-dot)]",
  muted: "bg-muted-foreground/40",
};

function FactoryLineRow({ row }: { row: FactoryLineRowModel }) {
  const activityId = row.lineId && row.lineId !== "unknown" ? `activity-line-run-${row.lineId}` : undefined;
  const inner = (
    <>
      <Sparkles className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <span className={cn("size-1.5 shrink-0 rounded-full", TONE_DOT_CLASS[row.tone])} aria-hidden />
      <span className="min-w-0 truncate text-[13px] tracking-[-0.01em] text-foreground">{row.lineName}</span>
    </>
  );
  const commonClass =
    "-mx-1.5 flex w-[calc(100%+0.75rem)] items-center gap-2 rounded-md px-1.5 py-1.5 text-left transition-colors hover:bg-accent/60 focus-visible:bg-accent/60";

  if (!activityId) {
    return <span className={cn(commonClass, "cursor-default")}>{inner}</span>;
  }

  return (
    <button
      type="button"
      className={commonClass}
      aria-label={`Show ${row.lineName} in activity`}
      onClick={() => document.getElementById(activityId)?.scrollIntoView({ behavior: "smooth", block: "start" })}
    >
      {inner}
    </button>
  );
}

function parseNumericFromString(value: string | undefined): number {
  if (!value) return 0;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatSpendingLine(totalTokens: number, totalCostCents: number): React.ReactNode {
  const tokens = totalTokens > 0 ? formatCompactTokens(totalTokens) : null;
  const usd = totalCostCents > 0 ? formatUsdCents(totalCostCents) : null;
  if (tokens && usd) {
    return (
      <>
        {tokens} <span className="text-muted-foreground">·</span> {usd}
      </>
    );
  }
  return tokens ?? usd ?? "";
}

function formatSpendingTooltip(totalTokens: number, totalCostCents: number): string {
  const parts: string[] = [];
  if (totalTokens > 0) parts.push(`${totalTokens.toLocaleString()} tokens`);
  if (totalCostCents > 0) parts.push(`$${(totalCostCents / 100).toFixed(2)}`);
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

function formatCompactDate(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  })
    .format(date)
    .replace(",", "");
}
