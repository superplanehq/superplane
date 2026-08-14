import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { cn } from "@/lib/utils";
import { Forward, UserPlus } from "lucide-react";
import { DispatchWorkOrderPopover } from "../DispatchWorkOrderPopover";
import { OrgUserReference } from "../OrgUserReference";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

/** Actions callable from any Work Orders layout (board card, list row, table row). */
export interface WorkOrderRowCallbacks {
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

interface AssigneeGroupProps {
  entry: WorkOrderListEntry;
  organizationId: string;
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
  /** Cap for the visible avatars before "+N" collapses the rest. */
  maxVisible?: number;
  size?: "sm" | "md";
}

/**
 * Small avatar stack with a `+N` overflow indicator and, when the caller
 * has permission, an inline picker to change assignees without opening
 * the detail page. Shared across board, list, and table.
 */
export function AssigneeGroup({
  entry,
  organizationId,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
  maxVisible = 1,
  size = "sm",
}: AssigneeGroupProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const assignees = entry.order.assignees ?? [];
  const visible = assignees.slice(0, maxVisible);
  const hiddenCount = Math.max(assignees.length - visible.length, 0);

  const stack = (
    <button
      type="button"
      className={cn(
        "inline-flex items-center gap-1 rounded-full transition-colors",
        canAssign ? "hover:bg-accent" : "cursor-default",
      )}
      aria-label="Change owners"
      data-testid={`work-order-row-assignees-${entry.id}`}
      disabled={!canAssign}
    >
      {assignees.length === 0 ? (
        <span
          className={cn(
            "inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-[11px] text-muted-foreground",
            canAssign && "hover:border-foreground hover:text-foreground",
          )}
        >
          <UserPlus className="size-3" aria-hidden />
          Assign
        </span>
      ) : (
        <span className="inline-flex items-center -space-x-1.5">
          {visible.map((assignee, index) => (
            <OrgUserReference
              key={assignee.id ?? index}
              display={resolveUser(assignee.id, assignee.name)}
              size={size}
              showName={false}
              className="rounded-full ring-2 ring-background"
            />
          ))}
          {hiddenCount > 0 ? (
            <span
              className="inline-flex items-center justify-center rounded-full bg-muted px-1.5 text-[10px] font-medium text-muted-foreground ring-2 ring-background"
              aria-label={`${hiddenCount} more owners`}
            >
              +{hiddenCount}
            </span>
          ) : null}
        </span>
      )}
    </button>
  );

  const trigger = canAssign ? (
    stack
  ) : (
    <Tooltip>
      <TooltipTrigger asChild>{stack}</TooltipTrigger>
      <TooltipContent>You don't have permission to update owners.</TooltipContent>
    </Tooltip>
  );

  if (!canAssign) {
    return (
      <div className="pointer-events-auto" onClick={(event) => event.stopPropagation()}>
        {trigger}
      </div>
    );
  }

  return (
    <div className="pointer-events-auto" onClick={(event) => event.stopPropagation()}>
      <WorkOrderAssigneesPopover
        organizationId={organizationId}
        selectedIds={entry.assigneeIds}
        canEdit={canAssign}
        isSaving={isAssigneesSaving}
        onSave={(ids) => onAssigneesSave(entry.id, ids)}
      >
        {trigger}
      </WorkOrderAssigneesPopover>
    </div>
  );
}

interface DispatchButtonProps {
  entry: WorkOrderListEntry;
  lines: FactoriesFactoryLine[];
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  /** Only draft/open work orders show the button. */
  visible: boolean;
  variant?: "ghost" | "outline";
}

export function InlineDispatchButton({
  entry,
  lines,
  canDispatch,
  isDispatching,
  onDispatch,
  visible,
  variant = "ghost",
}: DispatchButtonProps) {
  if (!visible) {
    return null;
  }
  return (
    <div className="pointer-events-auto" onClick={(event) => event.stopPropagation()}>
      <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
        <DispatchWorkOrderPopover
          lines={lines}
          isSaving={isDispatching}
          canDispatch={canDispatch}
          onDispatch={(input) => onDispatch(entry.id, input)}
        >
          <Button
            type="button"
            variant={variant}
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
            disabled={!canDispatch || lines.length === 0}
            aria-label="Dispatch to line"
            data-testid={`work-order-row-dispatch-${entry.id}`}
          >
            <Forward className="size-3.5" aria-hidden />
          </Button>
        </DispatchWorkOrderPopover>
      </PermissionTooltip>
    </div>
  );
}
