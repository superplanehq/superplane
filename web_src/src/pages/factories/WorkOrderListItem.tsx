import type { FactoriesWorkOrder } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/lib/date";
import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  CircleDashed,
  ClipboardList,
  Eye,
  Loader2,
  ShieldAlert,
  Wrench,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import { deriveWorkOrderProgress, getWorkOrderDisplayKey, type WorkOrderProgressPhase } from "./workOrderProgress";

const PHASE_ICON: Record<WorkOrderProgressPhase, LucideIcon> = {
  unassigned: CircleDashed,
  needs_attention: AlertCircle,
  no_plan: ClipboardList,
  planning: ClipboardList,
  implementation_in_progress: Wrench,
  verifications_running: Loader2,
  verifications_failed: ShieldAlert,
  ready_for_review: Eye,
  completed: CheckCircle2,
  rejected: XCircle,
};

const PHASE_ICON_CLASS: Partial<Record<WorkOrderProgressPhase, string>> = {
  needs_attention: "text-amber-500",
  verifications_failed: "text-red-500",
  verifications_running: "text-sky-400 animate-spin",
  implementation_in_progress: "text-violet-400",
  ready_for_review: "text-emerald-400",
  completed: "text-emerald-500",
  rejected: "text-red-400",
};

interface WorkOrderListItemProps {
  order: FactoriesWorkOrder;
  className?: string;
  canClaim?: boolean;
  isClaiming?: boolean;
  onClaim?: (orderId: string) => void;
}

export function WorkOrderListItem({
  order,
  className,
  canClaim = false,
  isClaiming = false,
  onClaim,
}: WorkOrderListItemProps) {
  const progress = deriveWorkOrderProgress(order);
  const Icon = PHASE_ICON[progress.phase];
  const updatedAt = order.updatedAt ?? order.createdAt;
  const timeLabel = updatedAt ? formatTimeAgo(new Date(updatedAt)) : "—";
  const showClaim = progress.phase === "unassigned" && onClaim && order.id;

  return (
    <div
      className={cn(
        "group flex items-start gap-3 px-4 py-3 transition-colors hover:bg-slate-50 dark:hover:bg-gray-800/60",
        className,
      )}
      data-testid="work-order-list-item"
    >
      <div className="mt-0.5 shrink-0">
        <Icon className={cn("h-4 w-4", PHASE_ICON_CLASS[progress.phase] ?? "text-slate-400 dark:text-gray-500")} />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-slate-500 dark:text-gray-500">{getWorkOrderDisplayKey(order)}</span>
          <p className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{order.title}</p>
          <Badge
            variant="outline"
            className="rounded-full px-2 py-0 text-[10px] font-medium text-slate-600 dark:border-gray-600 dark:text-gray-300"
          >
            {progress.stageLabel}
          </Badge>
        </div>
        {order.description ? (
          <p className="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">{order.description}</p>
        ) : (
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{progress.summary}</p>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-3 self-center text-xs text-gray-500 dark:text-gray-400">
        <span className="whitespace-nowrap">{timeLabel}</span>
        {showClaim ? (
          <PermissionTooltip allowed={canClaim} message="You don't have permission to claim work orders.">
            <LoadingButton
              type="button"
              size="sm"
              variant="outline"
              loading={isClaiming}
              loadingText="Claiming..."
              disabled={!canClaim || isClaiming}
              onClick={() => onClaim(order.id!)}
              data-testid="work-order-claim-button"
            >
              Claim
            </LoadingButton>
          </PermissionTooltip>
        ) : (
          <ChevronRight
            className="h-4 w-4 text-gray-400 opacity-0 transition-opacity group-hover:opacity-100 dark:text-gray-500"
            aria-hidden
          />
        )}
      </div>
    </div>
  );
}
