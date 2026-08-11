import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Check, Ellipsis, Link2 } from "lucide-react";
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
  displayStatus: WorkOrderDisplayStatus;
  isOpen: boolean;
  isDispatchable: boolean;
  isClosed: boolean;
  canClose: boolean;
  canManage: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  isUpdatingStatus: boolean;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
}

/**
 * Detail-page header. Renders just the title on the left and Copy link + a
 * kebab of lifecycle actions on the right. Status now lives in the sidebar
 * Overview and dispatch lives in the sidebar Factory Lines section, so this
 * header stays minimal.
 */
export function WorkOrderDetailHeader(props: WorkOrderDetailHeaderProps) {
  const { orderTitle } = props;
  return (
    <header className="sticky top-0 z-10 col-span-full mx-[calc(var(--workspace-page-gutter)*-1)] mb-3 bg-background px-[var(--workspace-page-gutter)] py-3">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <h1 className="workspace-page-title min-w-0 flex-1">{orderTitle}</h1>

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
      className="size-7 text-muted-foreground hover:bg-accent hover:text-foreground"
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
  // Back-to-draft transition is only safe when no line execution is active —
  // the `open → draft` FSM guard rejects otherwise, so we hide the action
  // instead of letting the API 400.
  const canReturnToDraft = isOpen && displayStatus !== "running";

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
            className="size-7 text-muted-foreground hover:bg-accent hover:text-foreground"
            disabled={disabled}
            aria-label="More actions"
            data-testid="work-order-actions-button"
          >
            <Ellipsis className="size-3.5" aria-hidden />
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
