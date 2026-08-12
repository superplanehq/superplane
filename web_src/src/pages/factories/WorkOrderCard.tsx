import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Forward, Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import { factoryWorkOrderRowClassName } from "./lib/factoryPageStyles";
import { factoryDetailPath, workOrderDetailPath } from "./lib/factoryPagePaths";
import { resolveWorkOrderCreatorDisplay } from "./lib/workOrderCreator";
import { OrgUserReference } from "./OrgUserReference";
import { WorkOrderExecutionsList } from "./WorkOrderExecutionsList";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "./lib/workOrderProgress";

interface WorkOrderCardProps {
  order: FactoriesWorkOrder;
  organizationId: string;
  factoryId: string;
  lines: FactoriesFactoryLine[];
  canDispatch?: boolean;
  isDispatching?: boolean;
  onDispatch?: (input: { lineName: string }) => Promise<void>;
}

export function WorkOrderCard({
  order,
  organizationId,
  factoryId,
  lines,
  canDispatch = false,
  isDispatching = false,
  onDispatch,
}: WorkOrderCardProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const displayStatus = getWorkOrderDisplayStatus(order);
  const statusMeta = getWorkOrderDisplayStatusMeta(displayStatus);
  const updatedAt = order.updatedAt ?? order.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
  const href = order.id
    ? workOrderDetailPath(organizationId, factoryId, order.id)
    : factoryDetailPath(organizationId, factoryId);
  const creatorDisplay = resolveWorkOrderCreatorDisplay(order.createdBy, resolveUser);
  const assigneeDisplays = (order.assignees ?? [])
    .filter((assignee) => assignee.id)
    .map((assignee) => resolveUser(assignee.id, assignee.name));
  const canShowDispatchButton = Boolean(
    (order.state === "STATE_OPEN" || order.state === "STATE_DRAFT") && onDispatch && order.id,
  );

  return (
    <article
      className={cn(
        factoryWorkOrderRowClassName,
        "relative -mx-6 cursor-pointer px-6 transition-colors hover:bg-accent/50",
      )}
      data-testid="work-order-card"
    >
      <Link to={href} className="absolute inset-0 z-0" aria-label={`Open ${order.title}`} />

      <div className="relative z-10 min-w-0 pointer-events-none">
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <Badge
              variant="outline"
              className={cn("inline-flex shrink-0 px-2.5 py-1 text-xs font-medium", statusMeta.className)}
            >
              {displayStatus === "running" ? (
                <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden />
              ) : null}
              {statusMeta.label}
            </Badge>
            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              <h3 className="workspace-section-title min-w-0">{order.title}</h3>
              <span className="shrink-0 text-xs text-muted-foreground">Updated {timeLabel}</span>
              {canShowDispatchButton && onDispatch ? (
                <WorkOrderCardDispatchButton
                  lines={lines}
                  canDispatch={canDispatch}
                  isDispatching={isDispatching}
                  onDispatch={onDispatch}
                />
              ) : null}
            </div>
          </div>

          <WorkOrderCardActors creatorDisplay={creatorDisplay} assigneeDisplays={assigneeDisplays} />
        </div>

        <WorkOrderExecutionsList
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={order.id}
          executions={order.executions}
          variant="compact"
        />
      </div>
    </article>
  );
}

function WorkOrderCardDispatchButton({
  lines,
  canDispatch,
  isDispatching,
  onDispatch,
}: {
  lines: FactoriesFactoryLine[];
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: (input: { lineName: string }) => Promise<void>;
}) {
  return (
    <div className="pointer-events-auto">
      <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
        <DispatchWorkOrderPopover
          lines={lines}
          isSaving={isDispatching}
          canDispatch={canDispatch}
          onDispatch={onDispatch}
        >
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground"
            disabled={!canDispatch || lines.length === 0}
            aria-label="Dispatch to line"
            data-testid="work-order-dispatch-button"
          >
            <Forward className="h-4 w-4" aria-hidden />
          </Button>
        </DispatchWorkOrderPopover>
      </PermissionTooltip>
    </div>
  );
}

function WorkOrderCardActors({
  creatorDisplay,
  assigneeDisplays,
}: {
  creatorDisplay: ReturnType<ReturnType<typeof useOrgUserLookup>["resolveUser"]>;
  assigneeDisplays: Array<ReturnType<ReturnType<typeof useOrgUserLookup>["resolveUser"]>>;
}) {
  return (
    <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      <span className="inline-flex items-center gap-1.5">
        Creator:
        <OrgUserReference display={creatorDisplay} size="md" showName={false} />
      </span>
      {assigneeDisplays.length > 0 ? (
        <span className="inline-flex items-center gap-1.5">
          Assignees:
          <span className="inline-flex items-center -space-x-1">
            {assigneeDisplays.map((display, index) => (
              <OrgUserReference
                key={display?.id ?? index}
                display={display}
                size="md"
                showName={false}
                className="rounded-full ring-2 ring-background"
              />
            ))}
          </span>
        </span>
      ) : null}
    </div>
  );
}
