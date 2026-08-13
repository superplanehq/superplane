import type { FactoriesAutomationRef, FactoriesWorkOrder } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import {
  Calendar,
  ChevronDown,
  CircleDollarSign,
  CircleDot,
  ExternalLink,
  Loader2,
  User,
  UserPlus,
} from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";

import { resolveWorkOrderCreatorDisplay } from "../lib/workOrderCreator";
import { formatWorkOrderDateTime } from "../lib/workOrderDateTime";
import type { WorkOrderDisplayStatus } from "../lib/workOrderProgress";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../lib/workOrderUsage";
import { OrgUserReference } from "../OrgUserReference";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";
import { OverviewRow, SidebarSectionHeading } from "./SidebarPrimitives";

interface WorkOrderSidebarOverviewProps {
  organizationId: string;
  order: FactoriesWorkOrder;
  displayStatus: WorkOrderDisplayStatus;
  statusMeta: { label: string; className: string };
  assigneeIds: string[];
  assigneeNames: string[];
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (assigneeIds: string[]) => Promise<void>;
}

export function WorkOrderSidebarOverview({
  organizationId,
  order,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
}: WorkOrderSidebarOverviewProps) {
  const createdAt = order.createdAt ? new Date(order.createdAt) : null;
  const totalTokens = parseWorkOrderMetric(order.totalTokens);
  const totalCostCents = parseWorkOrderMetric(order.totalCostCents);
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
            <span title={createdAt.toLocaleString()}>{formatWorkOrderDateTime(createdAt)}</span>
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
  const automation = order.createdBy?.automation;
  if (isAutomationRefResolved(automation)) {
    return <AutomationLink organizationId={organizationId} automation={automation} />;
  }
  const display = resolveWorkOrderCreatorDisplay(order.createdBy, resolveUser);
  if (display) {
    return <OrgUserReference display={display} size="xs" nameClassName="truncate text-[13px]" />;
  }
  return <span className="truncate">{order.createdBy?.user?.name ?? "Unknown"}</span>;
}

function isAutomationRefResolved(ref: FactoriesAutomationRef | undefined): ref is FactoriesAutomationRef {
  return Boolean(ref && (ref.nodeName || ref.appName));
}

export function AutomationLink({
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
        className="inline-flex min-w-0 max-w-full items-center gap-1 text-foreground underline underline-offset-2 hover:no-underline"
      >
        <span className="truncate">{label}</span>
        <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
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
          <Button
            type="button"
            variant="ghost"
            disabled={!canEdit || isSaving}
            aria-label={
              assigneeIds.length > 0 ? `Assignee: ${assigneeNames.filter(Boolean).join(", ")}` : "Assign work order"
            }
            className="-my-1.5 -mr-1.5 h-auto w-full min-w-0 justify-start gap-1.5 whitespace-normal rounded-md py-1.5 pr-1.5 pl-0 text-left text-[13px] tracking-[-0.01em] text-foreground hover:bg-accent/60 focus-visible:bg-accent/60"
            data-testid="work-order-edit-assignees"
          >
            <AssigneeButtonBody assigneeIds={assigneeIds} assigneeNames={assigneeNames} resolveUser={resolveUser} />
            <ChevronDown className="ml-auto size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </Button>
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

function formatSpendingLine(totalTokens: number, totalCostCents: number): ReactNode {
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
  if (totalCostCents > 0) parts.push(formatUsdCents(totalCostCents));
  return parts.join(" · ");
}
