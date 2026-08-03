import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Forward, Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import { factoryWorkOrderRowClassName } from "./factoryPageStyles";
import { WorkOrderExecutionsList } from "./WorkOrderExecutionsList";
import {
  getWorkOrderDisplayStatus,
  getWorkOrderDisplayStatusMeta,
} from "./workOrderProgress";

interface WorkOrderCardProps {
  order: FactoriesWorkOrder;
  factoryHref: string;
  organizationId: string;
  lines: FactoriesFactoryLine[];
  canDispatch?: boolean;
  isDispatching?: boolean;
  onDispatch?: (lineName: string) => Promise<void>;
}

export function WorkOrderCard({
  order,
  factoryHref,
  organizationId,
  lines,
  canDispatch = false,
  isDispatching = false,
  onDispatch,
}: WorkOrderCardProps) {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const statusMeta = getWorkOrderDisplayStatusMeta(displayStatus);
  const updatedAt = order.updatedAt ?? order.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
  const href = order.id ? `${factoryHref}/orders/${order.id}` : factoryHref;
  const authorLabel = order.createdBy?.name?.trim() || "Unknown";
  const showDispatch = order.state === "STATE_OPEN" && onDispatch && order.id;

  return (
    <article
      className={cn(
        factoryWorkOrderRowClassName,
        "relative -mx-6 cursor-pointer px-6 transition-colors hover:bg-gray-50 dark:hover:bg-gray-800/40",
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
            <div className="flex min-w-0 flex-wrap items-center gap-1">
              <h3 className="min-w-0 text-base font-semibold text-gray-900 dark:text-gray-100">{order.title}</h3>
              {showDispatch ? (
                <div className="pointer-events-auto">
                  <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
                    <DispatchWorkOrderPopover
                      lines={lines}
                      isSaving={isDispatching}
                      canDispatch={canDispatch}
                      onDispatch={onDispatch!}
                    >
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 shrink-0 text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
                        disabled={!canDispatch || lines.length === 0}
                        aria-label="Dispatch to line"
                        data-testid="work-order-dispatch-button"
                      >
                        <Forward className="h-4 w-4" aria-hidden />
                      </Button>
                    </DispatchWorkOrderPopover>
                  </PermissionTooltip>
                </div>
              ) : null}
            </div>
          </div>

          <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
            <span>Created by {authorLabel}</span>
            <span>Updated {timeLabel}</span>
          </div>
        </div>

        <WorkOrderExecutionsList organizationId={organizationId} executions={order.executions} variant="compact" />
      </div>
    </article>
  );
}
