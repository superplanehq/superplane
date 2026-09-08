import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Ellipsis } from "lucide-react";
import { Fragment } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import {
  applyWorkOrderStatusAction,
  type WorkOrderStatusAction,
  type WorkOrderStatusActionKind,
} from "./lib/workOrderStatusActions";
import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";

const STATUS_ACTION_TEST_ID: Record<WorkOrderStatusActionKind, string> = {
  complete: "work-order-complete-button",
  reject: "work-order-reject-button",
  "reject-draft": "work-order-reject-draft-button",
  "back-to-draft": "work-order-back-to-draft-button",
  reopen: "work-order-reopen-open-button",
};

export interface WorkOrderOverflowMenuProps {
  canCreate: boolean;
  isDuplicating?: boolean;
  onDuplicate: () => void;
  actions: WorkOrderStatusAction[];
  onClose: (result: FactoriesWorkOrderResult) => void;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  disabled?: boolean;
  triggerClassName?: string;
  triggerTestId?: string;
}

export function WorkOrderOverflowMenu({
  canCreate,
  isDuplicating = false,
  onDuplicate,
  actions,
  onClose,
  onStatusChange,
  disabled = false,
  triggerClassName = "text-muted-foreground hover:bg-accent hover:text-foreground",
  triggerTestId = "work-order-actions-button",
}: WorkOrderOverflowMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className={triggerClassName}
          disabled={disabled}
          aria-label="More actions"
          data-testid={triggerTestId}
        >
          <Ellipsis className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-48">
        <PermissionTooltip allowed={canCreate} message="You don't have permission to create tasks.">
          <DropdownMenuItem
            disabled={!canCreate || isDuplicating}
            onSelect={() => onDuplicate()}
            data-testid="work-order-duplicate-button"
          >
            Duplicate
          </DropdownMenuItem>
        </PermissionTooltip>
        {actions.map((action, index) => (
          <Fragment key={action.kind}>
            {index === 0 || action.separatorBefore ? <DropdownMenuSeparator /> : null}
            <DropdownMenuItem
              disabled={action.disabled}
              onSelect={() => applyWorkOrderStatusAction(action.kind, { onClose, onStatusChange })}
              data-testid={STATUS_ACTION_TEST_ID[action.kind]}
            >
              {action.label}
            </DropdownMenuItem>
          </Fragment>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
