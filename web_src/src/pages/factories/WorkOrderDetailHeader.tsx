import type { FactoriesFactoryLine, FactoriesWorkOrderResult } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { Forward, Loader2 } from "lucide-react";
import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

interface WorkOrderDetailHeaderProps {
  orderTitle: string;
  statusMeta: { label: string; className: string };
  displayStatus: WorkOrderDisplayStatus;
  isOpen: boolean;
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canClose: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  onDispatch: (lineName: string) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
}

export function WorkOrderDetailHeader({
  orderTitle,
  statusMeta,
  displayStatus,
  isOpen,
  factoryLines,
  canDispatch,
  canClose,
  permissionsLoading,
  isDispatching,
  isCompleting,
  isRejecting,
  isClosing,
  onDispatch,
  onClose,
}: WorkOrderDetailHeaderProps) {
  return (
    <header className="border-b border-gray-200 pb-6 dark:border-gray-700/70">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
          <Badge
            variant="outline"
            className={cn("inline-flex shrink-0 px-2.5 py-1 text-xs font-medium", statusMeta.className)}
          >
            {displayStatus === "running" ? <Loader2 className="mr-1 inline h-3 w-3 animate-spin" aria-hidden /> : null}
            {statusMeta.label}
          </Badge>
          <h1 className="min-w-0 text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
            {orderTitle}
          </h1>
        </div>

        {isOpen ? (
          <div className="flex flex-wrap items-center gap-2">
            <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
              <DispatchWorkOrderPopover
                lines={factoryLines}
                isSaving={isDispatching}
                canDispatch={canDispatch || permissionsLoading}
                onDispatch={onDispatch}
              >
                <Button
                  type="button"
                  disabled={!canDispatch || factoryLines.length === 0}
                  data-testid="work-order-dispatch-button"
                >
                  Dispatch
                  <Forward className="ml-1.5 h-4 w-4" aria-hidden />
                </Button>
              </DispatchWorkOrderPopover>
            </PermissionTooltip>

            <PermissionTooltip allowed={canClose} message="You don't have permission to close work orders.">
              <LoadingButton
                type="button"
                variant="ghost"
                disabled={!canClose || isClosing}
                loading={isCompleting}
                loadingText="Completing..."
                onClick={() => void onClose("RESULT_COMPLETED")}
                data-testid="work-order-complete-button"
              >
                Complete
              </LoadingButton>
            </PermissionTooltip>

            <PermissionTooltip allowed={canClose} message="You don't have permission to close work orders.">
              <LoadingButton
                type="button"
                variant="ghost"
                disabled={!canClose || isClosing}
                loading={isRejecting}
                loadingText="Rejecting..."
                onClick={() => void onClose("RESULT_REJECTED")}
                data-testid="work-order-reject-button"
              >
                Reject
              </LoadingButton>
            </PermissionTooltip>
          </div>
        ) : null}
      </div>
    </header>
  );
}
