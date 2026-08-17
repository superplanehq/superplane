import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Check, Ellipsis, Link2 } from "lucide-react";
import { Fragment, useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { WorkspacePageHeader } from "./layout/WorkspacePageHeader";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";
import { useWorkOrderReactionsSlot } from "./workOrderReactionsSlot";

interface WorkOrderDetailHeaderProps {
  orderTitle: string;
  /** Short identifier (e.g. `SP-42`). Rendered as a kicker above the title. */
  orderIdentifier?: string;
  /** Work order id, passed to the Storybook-only reactions slot. */
  orderId: string;
  /** Back link target (Work Orders list). */
  backHref: string;
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

export function WorkOrderDetailHeader(props: WorkOrderDetailHeaderProps) {
  const ReactionsSlot = useWorkOrderReactionsSlot();

  return (
    <WorkspacePageHeader
      variant="entity"
      backHref={props.backHref}
      backLabel="Work Orders"
      backTestId="work-order-detail-back"
      kicker={props.orderIdentifier}
      title={props.orderTitle}
      actions={
        <>
          <CopyLinkButton />
          <HeaderOverflowMenu {...props} />
        </>
      }
      belowRow={
        ReactionsSlot ? (
          <div className="mt-3" data-testid="work-order-reactions-slot">
            <ReactionsSlot workOrderId={props.orderId} />
          </div>
        ) : null
      }
    />
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
      size="icon-xs"
      onClick={() => void handleCopy()}
      className="text-muted-foreground hover:bg-accent hover:text-foreground"
      aria-label="Copy link to work order"
      data-testid="work-order-copy-link-button"
    >
      {copied ? <Check className="size-3.5" aria-hidden /> : <Link2 className="size-3.5" aria-hidden />}
    </Button>
  );
}

function HeaderOverflowMenu(props: WorkOrderDetailHeaderProps) {
  const actions = buildHeaderActions(props);
  if (actions.length === 0) {
    return null;
  }

  const disabled = props.isClosing || props.isUpdatingStatus || props.isCompleting || props.isRejecting;

  return (
    <DropdownMenu>
      <PermissionTooltip
        allowed={props.canClose || props.canManage}
        message="You don't have permission to manage this work order."
      >
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:bg-accent hover:text-foreground"
            disabled={disabled}
            aria-label="More actions"
            data-testid="work-order-actions-button"
          >
            <Ellipsis className="size-3.5" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
      </PermissionTooltip>

      <DropdownMenuContent align="end" className="w-48">
        {actions.map((action) => (
          <Fragment key={action.testId}>
            {action.separatorBefore ? <DropdownMenuSeparator /> : null}
            <DropdownMenuItem disabled={action.disabled} onSelect={action.onSelect} data-testid={action.testId}>
              {action.label}
            </DropdownMenuItem>
          </Fragment>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface HeaderAction {
  label: string;
  testId: string;
  disabled: boolean;
  separatorBefore?: boolean;
  onSelect: () => void;
}

function buildHeaderActions(props: WorkOrderDetailHeaderProps): HeaderAction[] {
  const actions: HeaderAction[] = [];
  const isDraft = props.isDispatchable && !props.isOpen && !props.isClosed;

  if (props.isOpen) {
    actions.push(
      closeAction("Complete", "RESULT_COMPLETED", "work-order-complete-button", props),
      closeAction("Reject", "RESULT_REJECTED", "work-order-reject-button", props),
    );
  }

  if (props.isOpen && props.displayStatus !== "running") {
    actions.push({
      label: "Back to draft",
      testId: "work-order-back-to-draft-button",
      disabled: !props.canManage || props.isUpdatingStatus,
      separatorBefore: true,
      onSelect: () => void props.onStatusChange("STATE_DRAFT"),
    });
  }

  if (isDraft) {
    actions.push(closeAction("Reject", "RESULT_REJECTED", "work-order-reject-draft-button", props));
  }

  if (props.isClosed) {
    actions.push({
      label: "Reopen",
      testId: "work-order-reopen-open-button",
      disabled: !props.canManage || props.isUpdatingStatus,
      onSelect: () => void props.onStatusChange("STATE_OPEN"),
    });
  }

  return actions;
}

function closeAction(
  label: string,
  result: FactoriesWorkOrderResult,
  testId: string,
  props: WorkOrderDetailHeaderProps,
): HeaderAction {
  return {
    label,
    testId,
    disabled: !props.canClose || props.isClosing,
    onSelect: () => props.onClose(result),
  };
}
