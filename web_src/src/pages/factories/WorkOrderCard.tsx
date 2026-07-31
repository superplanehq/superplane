import type { FactoriesWorkOrder } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { Loader2, Send } from "lucide-react";
import { Link } from "react-router-dom";
import { factoryWorkOrderRowClassName } from "./factoryPageStyles";
import { deriveWorkOrderProgress, getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

interface WorkOrderCardProps {
  order: FactoriesWorkOrder;
  factoryHref: string;
  createdByLabel?: string;
  pipelineLabels?: string[];
  canDispatch?: boolean;
  onDispatch?: (orderId: string) => void;
}

export function WorkOrderCard({
  order,
  factoryHref,
  createdByLabel,
  pipelineLabels = [],
  canDispatch = false,
  onDispatch,
}: WorkOrderCardProps) {
  const progress = deriveWorkOrderProgress(order);
  const displayStatus = getWorkOrderDisplayStatus(order);
  const statusMeta = getWorkOrderDisplayStatusMeta(displayStatus);
  const updatedAt = order.updatedAt ?? order.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
  const href = order.id ? `${factoryHref}/orders/${order.id}` : factoryHref;
  const assigneeName = order.assignees?.[0]?.name;
  const authorLabel = createdByLabel ?? assigneeName ?? "Unknown";
  const showDispatch = order.state === "STATE_OPEN" && onDispatch && order.id;

  return (
    <article className={factoryWorkOrderRowClassName} data-testid="work-order-card">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <Link to={href} className="block no-underline">
            <h3 className="text-base font-semibold text-gray-900 group-hover:underline dark:text-gray-100">
              {order.title}
            </h3>
          </Link>
          {order.description ? (
            <p className="mt-2 line-clamp-2 text-sm leading-relaxed text-gray-500 dark:text-gray-400">
              {order.description}
            </p>
          ) : (
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{progress.summary}</p>
          )}

          {pipelineLabels.length > 0 ? (
            <ul className="mt-4 space-y-1.5">
              {pipelineLabels.map((label) => (
                <li key={label} className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                  <span className="inline-flex h-4 w-4 items-center justify-center rounded-full border border-slate-300 dark:border-gray-600">
                    {displayStatus === "running" ? (
                      <Loader2 className="h-2.5 w-2.5 animate-spin text-violet-500" aria-hidden />
                    ) : (
                      <span className="h-1.5 w-1.5 rounded-full bg-slate-300 dark:bg-gray-600" aria-hidden />
                    )}
                  </span>
                  <span>{label}</span>
                  <span className="text-gray-400 dark:text-gray-500">
                    {displayStatus === "running" ? "Running" : displayStatus === "successful" ? "Done" : "Planned"}
                  </span>
                </li>
              ))}
            </ul>
          ) : null}
        </div>

        <Badge
          variant="outline"
          className={cn("shrink-0 rounded-full px-2.5 py-1 text-xs font-medium", statusMeta.className)}
        >
          {displayStatus === "running" ? <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden /> : null}
          {statusMeta.label}
        </Badge>
      </div>

      <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
          <span>Created by {authorLabel}</span>
          <span>Updated {timeLabel}</span>
        </div>

        {showDispatch ? (
          <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!canDispatch}
              onClick={() => onDispatch(order.id!)}
              data-testid="work-order-dispatch-button"
            >
              <Send className="mr-1.5 h-3.5 w-3.5" aria-hidden />
              Dispatch
            </Button>
          </PermissionTooltip>
        ) : null}
      </div>
    </article>
  );
}
