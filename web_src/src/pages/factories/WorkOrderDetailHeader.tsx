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
import {
  applyWorkOrderStatusAction,
  buildWorkOrderStatusActions,
  type WorkOrderStatusActionKind,
} from "./lib/workOrderStatusActions";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";

interface WorkOrderDetailHeaderProps {
  orderTitle: string;
  /** Short identifier (e.g. `SP-42`). Rendered as a kicker above the title. */
  orderIdentifier?: string;
  /** Back link target (Work Orders list). Omit in the card dialog. */
  backHref?: string;
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
  className?: string;
}

export function WorkOrderDetailHeader(props: WorkOrderDetailHeaderProps) {
  return (
    <WorkspacePageHeader
      className={props.className}
      variant="entity"
      backHref={props.backHref}
      backLabel={props.backHref ? "Work Orders" : undefined}
      backTestId="work-order-detail-back"
      kicker={props.orderIdentifier}
      title={props.orderTitle}
      actions={
        <>
          <CopyLinkButton />
          <HeaderOverflowMenu {...props} />
        </>
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

const HEADER_ACTION_TEST_ID: Record<WorkOrderStatusActionKind, string> = {
  complete: "work-order-complete-button",
  reject: "work-order-reject-button",
  "reject-draft": "work-order-reject-draft-button",
  "back-to-draft": "work-order-back-to-draft-button",
  reopen: "work-order-reopen-open-button",
};

function HeaderOverflowMenu(props: WorkOrderDetailHeaderProps) {
  const actions = buildWorkOrderStatusActions(props);
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
          <Fragment key={action.kind}>
            {action.separatorBefore ? <DropdownMenuSeparator /> : null}
            <DropdownMenuItem
              disabled={action.disabled}
              onSelect={() => applyWorkOrderStatusAction(action.kind, props)}
              data-testid={HEADER_ACTION_TEST_ID[action.kind]}
            >
              {action.label}
            </DropdownMenuItem>
          </Fragment>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
