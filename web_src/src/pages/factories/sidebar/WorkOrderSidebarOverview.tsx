import type { FactoriesAutomationRef, FactoriesWorkOrder } from "@/api-client";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { factoryAppPath } from "../lib/factoryPagePaths";
import { cn } from "@/lib/utils";
import { Calendar, CircleDollarSign, CircleDot, ExternalLink, Loader2, User, UserPlus } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";

import { resolveWorkOrderCreatorDisplay } from "../lib/workOrderCreator";
import { formatWorkOrderDateTime } from "../lib/workOrderDateTime";
import type { WorkOrderDisplayStatus } from "../lib/workOrderProgress";
import {
  formatCompactTokens,
  formatDurationSeconds,
  formatUsdCents,
  parseWorkOrderMetric,
} from "../lib/workOrderUsage";
import { OrgUserReference } from "../OrgUserReference";
import { OverviewRow, SidebarSectionHeading } from "./SidebarPrimitives";
import { useWorkOrderOverviewMissionSlot } from "./workOrderOverviewSlots";

interface WorkOrderSidebarOverviewProps {
  organizationId: string;
  factoryKey: string;
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
  factoryKey,
  order,
  displayStatus,
  statusMeta,
  assigneeIds,
  assigneeNames,
}: WorkOrderSidebarOverviewProps) {
  const createdAt = order.createdAt ? new Date(order.createdAt) : null;
  const totalTokens = parseWorkOrderMetric(order.totalTokens);
  const totalCostCents = parseWorkOrderMetric(order.totalCostCents);
  const durationSeconds = parseWorkOrderMetric(order.totalDurationSeconds);
  const showSpending = totalTokens > 0 || totalCostCents > 0 || durationSeconds > 0;
  const MissionSlot = useWorkOrderOverviewMissionSlot();

  return (
    <section>
      <SidebarSectionHeading>Overview</SidebarSectionHeading>
      <div className="mt-2">
        <OverviewRow icon={<CircleDot className="size-3.5" aria-hidden />} srLabel="Status">
          <StatusValue displayStatus={displayStatus} label={statusMeta.label} />
        </OverviewRow>

        <OverviewRow icon={<UserPlus className="size-3.5" aria-hidden />} srLabel="Author">
          <CreatorValue organizationId={organizationId} factoryKey={factoryKey} order={order} />
        </OverviewRow>

        <AssigneeOverviewRow organizationId={organizationId} assigneeIds={assigneeIds} assigneeNames={assigneeNames} />

        {MissionSlot ? <MissionSlot workOrderId={order.id ?? ""} /> : null}

        {createdAt ? (
          <OverviewRow icon={<Calendar className="size-3.5" aria-hidden />} srLabel="Created">
            <span title={createdAt.toLocaleString()}>{formatWorkOrderDateTime(createdAt)}</span>
          </OverviewRow>
        ) : null}

        {showSpending ? (
          <OverviewRow icon={<CircleDollarSign className="size-3.5" aria-hidden />} srLabel="Spending">
            <span title={formatSpendingTooltip(totalTokens, totalCostCents, durationSeconds)}>
              {formatSpendingLine(totalTokens, totalCostCents, durationSeconds)}
            </span>
          </OverviewRow>
        ) : null}
      </div>
    </section>
  );
}

const STATUS_TEXT_CLASSNAME: Record<WorkOrderDisplayStatus, string> = {
  completed: "text-[color:var(--status-completed-fg)]",
  running: "text-[color:var(--status-running-fg)]",
  failed: "text-[color:var(--status-failed-fg)]",
  rejected: "text-[color:var(--status-failed-fg)]",
  cancelled: "text-[color:var(--status-cancelled-fg)]",
  waiting: "text-[color:var(--status-waiting-fg)]",
  draft: "text-[color:var(--status-draft-fg)]",
};

const STATUS_DOT_CLASSNAME: Record<WorkOrderDisplayStatus, string> = {
  completed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
  rejected: "bg-[color:var(--status-failed-dot)]",
  cancelled: "bg-[color:var(--status-cancelled-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  draft: "bg-[color:var(--status-draft-dot)]",
};

function StatusValue({ displayStatus, label }: { displayStatus: WorkOrderDisplayStatus; label: string }) {
  const textClass = STATUS_TEXT_CLASSNAME[displayStatus];
  const dotClass = STATUS_DOT_CLASSNAME[displayStatus];
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

function CreatorValue({
  organizationId,
  factoryKey,
  order,
}: {
  organizationId: string;
  factoryKey: string;
  order: FactoriesWorkOrder;
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const automation = order.createdBy?.automation;
  if (isAutomationRefResolved(automation)) {
    return <AutomationLink organizationId={organizationId} factoryKey={factoryKey} automation={automation} />;
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
  factoryKey,
  automation,
}: {
  organizationId: string;
  factoryKey: string;
  automation: FactoriesAutomationRef;
}) {
  const label = automation.nodeName || automation.appName || "Automation";
  if (automation.appId) {
    return (
      <Link
        to={factoryAppPath(organizationId, factoryKey, automation.appId)}
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
}: {
  organizationId: string;
  assigneeIds: string[];
  assigneeNames: string[];
}) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  return (
    <OverviewRow icon={<User className="size-3.5" aria-hidden />} srLabel="Owner">
      <span data-testid="work-order-edit-assignees">
        <AssigneeButtonBody assigneeIds={assigneeIds} assigneeNames={assigneeNames} resolveUser={resolveUser} />
      </span>
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
    return <span className="min-w-0 truncate text-muted-foreground">No owner</span>;
  }
  const display = resolveUser(assigneeIds[0], assigneeNames[0]);
  return <OrgUserReference display={display} size="xs" nameClassName="truncate text-[13px]" />;
}

function formatSpendingLine(totalTokens: number, totalCostCents: number, durationSeconds: number): ReactNode {
  const parts: ReactNode[] = [];
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
  }
  if (totalTokens > 0) {
    parts.push(formatCompactTokens(totalTokens));
  }
  if (durationSeconds > 0) {
    parts.push(formatDurationSeconds(durationSeconds));
  }
  return parts.reduce<ReactNode>((acc, part, index) => {
    if (index === 0) {
      return part;
    }
    return (
      <>
        {acc} <span className="text-muted-foreground">·</span> {part}
      </>
    );
  }, "");
}

function formatSpendingTooltip(totalTokens: number, totalCostCents: number, durationSeconds: number): string {
  const parts: string[] = [];
  if (totalCostCents > 0) parts.push(formatUsdCents(totalCostCents));
  if (totalTokens > 0) parts.push(`${totalTokens.toLocaleString()} tokens`);
  if (durationSeconds > 0) parts.push(formatDurationSeconds(durationSeconds));
  return parts.join(" · ");
}
