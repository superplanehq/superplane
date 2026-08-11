import type { FactoriesFactoryLine, FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Check, Link2, Loader2, MoreHorizontal } from "lucide-react";
import { useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailHeaderProps {
  orderTitle: string;
  statusMeta: { label: string; className: string };
  displayStatus: WorkOrderDisplayStatus;
  isOpen: boolean;
  isDispatchable: boolean;
  isClosed: boolean;
  // factoryLines is still required so the props shape stays stable for
  // stories/tests. It's unused now that dispatch moved into the sidebar.
  factoryLines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canClose: boolean;
  canManage: boolean;
  permissionsLoading: boolean;
  isDispatching: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  isUpdatingStatus: boolean;
  onDispatch: (input: { lineName: string; note?: string }) => Promise<void>;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
}

export function WorkOrderDetailHeader(props: WorkOrderDetailHeaderProps) {
  const { orderTitle, statusMeta, displayStatus } = props;
  return (
    <header className="pb-6">
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

        <div className="flex flex-wrap items-center gap-1">
          <CopyLinkButton />
          <HeaderOverflowMenu {...props} />
        </div>
      </div>
    </header>
  );
}

function CopyLinkButton() {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      showSuccessToast("Link copied to clipboard.");
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      showErrorToast("Failed to copy link.");
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      onClick={() => void handleCopy()}
      className="h-8 w-8 text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
      aria-label="Copy link to work order"
      data-testid="work-order-copy-link-button"
    >
      {copied ? <Check className="h-4 w-4" aria-hidden /> : <Link2 className="h-4 w-4" aria-hidden />}
    </Button>
  );
}

function HeaderOverflowMenu({
  displayStatus,
  isOpen,
  isDispatchable,
  isClosed,
  canClose,
  canManage,
  isCompleting,
  isRejecting,
  isClosing,
  isUpdatingStatus,
  onClose,
  onStatusChange,
}: WorkOrderDetailHeaderProps) {
  // Draft is "dispatchable, not yet open, not closed" — the only lifecycle
  // stage where an operator can abandon the order before any work runs.
  const isDraft = isDispatchable && !isOpen && !isClosed;
  // Back-to-draft transition is only safe when no line execution is active
  // AND no plan approval is pending — the `open → draft` FSM guard rejects
  // both cases, so we hide the action instead of letting the API 400.
  const canReturnToDraft = isOpen && displayStatus !== "running" && displayStatus !== "waiting";

  const anyActionAvailable = isDraft || isOpen || isClosed;
  if (!anyActionAvailable) {
    return null;
  }

  const disabled = isClosing || isUpdatingStatus || isCompleting || isRejecting;

  return (
    <DropdownMenu>
      <PermissionTooltip allowed={canClose || canManage} message="You don't have permission to manage this work order.">
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
            disabled={disabled}
            aria-label="More actions"
            data-testid="work-order-actions-button"
          >
            <MoreHorizontal className="h-4 w-4" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
      </PermissionTooltip>

      <DropdownMenuContent align="end" className="w-48">
        {isOpen ? (
          <>
            <DropdownMenuItem
              disabled={!canClose || isClosing}
              onSelect={() => onClose("RESULT_COMPLETED")}
              data-testid="work-order-complete-button"
            >
              Complete
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={!canClose || isClosing}
              onSelect={() => onClose("RESULT_REJECTED")}
              data-testid="work-order-reject-button"
            >
              Reject
            </DropdownMenuItem>
            {canReturnToDraft ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  disabled={!canManage || isUpdatingStatus}
                  onSelect={() => void onStatusChange("STATE_DRAFT")}
                  data-testid="work-order-back-to-draft-button"
                >
                  Back to draft
                </DropdownMenuItem>
              </>
            ) : null}
          </>
        ) : null}

        {isDraft ? (
          <DropdownMenuItem
            disabled={!canClose || isClosing}
            onSelect={() => onClose("RESULT_REJECTED")}
            data-testid="work-order-reject-draft-button"
          >
            Reject
          </DropdownMenuItem>
        ) : null}

        {isClosed ? (
          <DropdownMenuItem
            disabled={!canManage || isUpdatingStatus}
            onSelect={() => void onStatusChange("STATE_OPEN")}
            data-testid="work-order-reopen-open-button"
          >
            Reopen
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
