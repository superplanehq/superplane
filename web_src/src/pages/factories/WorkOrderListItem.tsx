import type { FactoriesWorkOrder } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import { CheckCircle2, ChevronRight, Loader2, PlayCircle, XCircle, type LucideIcon } from "lucide-react";
import {
  getWorkOrderDisplayKey,
  getWorkOrderDisplayStatus,
  getWorkOrderDisplayStatusMeta,
  getWorkOrderStatusSummary,
  type WorkOrderDisplayStatus,
} from "./workOrderProgress";

const STATUS_ICON: Record<WorkOrderDisplayStatus, LucideIcon> = {
  open: PlayCircle,
  running: Loader2,
  failed: XCircle,
  completed: CheckCircle2,
  rejected: XCircle,
};

const STATUS_ICON_CLASS: Partial<Record<WorkOrderDisplayStatus, string>> = {
  open: "text-sky-500",
  running: "text-violet-400 animate-spin",
  failed: "text-red-500",
  completed: "text-emerald-500",
  rejected: "text-gray-400",
};

interface WorkOrderListItemProps {
  order: FactoriesWorkOrder;
  className?: string;
}

export function WorkOrderListItem({ order, className }: WorkOrderListItemProps) {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const statusMeta = getWorkOrderDisplayStatusMeta(displayStatus);
  const Icon = STATUS_ICON[displayStatus];
  const updatedAt = order.updatedAt ?? order.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";

  return (
    <div
      className={cn(
        "group flex items-start gap-3 px-4 py-3 transition-colors hover:bg-slate-50 dark:hover:bg-gray-800/60",
        className,
      )}
      data-testid="work-order-list-item"
    >
      <div className="mt-0.5 shrink-0">
        <Icon className={cn("h-4 w-4", STATUS_ICON_CLASS[displayStatus] ?? "text-slate-400 dark:text-gray-500")} />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-slate-500 dark:text-gray-500">{getWorkOrderDisplayKey(order)}</span>
          <p className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{order.title}</p>
          <Badge
            variant="outline"
            className={cn("rounded-full px-2 py-0 text-[10px] font-medium", statusMeta.className)}
          >
            {statusMeta.label}
          </Badge>
        </div>
        {order.description ? (
          <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">{order.description}</p>
        ) : (
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{getWorkOrderStatusSummary(order)}</p>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-3 self-center text-xs text-gray-500 dark:text-gray-400">
        <span className="whitespace-nowrap">{timeLabel}</span>
        <ChevronRight
          className="h-4 w-4 text-gray-400 opacity-0 transition-opacity group-hover:opacity-100 dark:text-gray-500"
          aria-hidden
        />
      </div>
    </div>
  );
}
