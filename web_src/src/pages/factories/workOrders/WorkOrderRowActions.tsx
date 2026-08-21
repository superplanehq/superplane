import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { cn } from "@/lib/utils";
import { ArrowRight, Forward, Loader2, UserPlus } from "lucide-react";
import { DispatchWorkOrderPopover } from "../DispatchWorkOrderPopover";
import { OrgUserReference } from "../OrgUserReference";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

/** Actions callable from list and table rows. Cards keep owner changes only. */
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
  size?: "sm" | "md";
}

/**
 * Single owner avatar, or an Assign chip when nobody owns the work order.
 * The picker replaces the owner; a work order cannot have two people.
 */
export function AssigneeGroup({
  entry,
  organizationId,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
  size = "sm",
}: AssigneeGroupProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const owner = entry.order.assignees?.[0];

  const stack = (
    <button
      type="button"
      className={cn(
        "inline-flex items-center gap-1 rounded-full transition-colors",
        canAssign ? "hover:bg-accent" : "cursor-default",
      )}
      aria-label="Change owner"
      data-testid={`work-order-row-assignees-${entry.id}`}
      disabled={!canAssign}
    >
      {owner ? (
        <OrgUserReference
          display={resolveUser(owner.id, owner.name)}
          size={size}
          showName={false}
          className="rounded-full ring-2 ring-background"
        />
      ) : (
        <span
          className={cn(
            "inline-flex items-center gap-1 rounded-full border border-dashed border-border px-2 py-0.5 text-[11px] text-muted-foreground",
            canAssign && "hover:border-foreground hover:text-foreground",
          )}
        >
          <UserPlus className="size-3" aria-hidden />
          Assign
        </span>
      )}
    </button>
  );

  const trigger = canAssign ? (
    stack
  ) : (
    <Tooltip>
      <TooltipTrigger asChild>{stack}</TooltipTrigger>
      <TooltipContent>You don't have permission to update the owner.</TooltipContent>
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

/** Line to start on: the preferred line when it exists, else the only line. */
export function resolveStartLineName(lines: FactoriesFactoryLine[], preferredLineName?: string): string | undefined {
  const names = lines.map((line) => line.name?.trim()).filter((name): name is string => Boolean(name));
  if (preferredLineName && names.includes(preferredLineName)) {
    return preferredLineName;
  }
  if (names.length === 1) {
    return names[0];
  }
  return undefined;
}

interface StartDraftButtonProps {
  entry: WorkOrderListEntry;
  lines: FactoriesFactoryLine[];
  preferredLineName?: string;
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
}

/**
 * Right-quarter hover control that starts a draft. The panel is a chevron
 * pointing into the card; the arrow means Start.
 */
export function StartDraftButton({
  entry,
  lines,
  preferredLineName,
  canDispatch,
  isDispatching,
  onDispatch,
}: StartDraftButtonProps) {
  if (entry.displayStatus !== "draft") {
    return null;
  }

  const lineName = resolveStartLineName(lines, preferredLineName);
  const disabled = !canDispatch || lines.length === 0;

  const startButton = (
    <button
      type="button"
      aria-label="Start"
      disabled={disabled || isDispatching}
      data-testid={`work-order-card-start-${entry.id}`}
      className={cn(
        "flex h-full w-full items-center justify-end pr-[18%] text-primary-foreground",
        "bg-primary transition-[filter] duration-200 hover:brightness-110",
        "[clip-path:polygon(38%_0,100%_0,100%_100%,38%_100%,0_50%)]",
        (disabled || isDispatching) && "opacity-90",
      )}
      onClick={lineName ? () => void onDispatch(entry.id, { lineName }) : undefined}
    >
      {isDispatching ? (
        <Loader2 className="size-4 animate-spin" aria-hidden />
      ) : (
        <ArrowRight
          className="size-4 translate-x-0 transition-transform duration-200 group-hover:translate-x-0.5"
          aria-hidden
        />
      )}
    </button>
  );

  return (
    <div
      className={cn(
        "absolute inset-y-0 right-0 z-20 w-1/4 overflow-hidden rounded-r-md transition-opacity duration-200",
        "pointer-events-none opacity-0 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100",
        isDispatching && "pointer-events-auto opacity-100",
      )}
      data-testid={`work-order-card-hover-actions-${entry.id}`}
      onClick={(event) => event.stopPropagation()}
    >
      <PermissionTooltip allowed={canDispatch} message="You don't have permission to start this work order.">
        {lineName ? (
          startButton
        ) : (
          <DispatchWorkOrderPopover
            lines={lines}
            isSaving={isDispatching}
            canDispatch={canDispatch}
            submitLabel="Start"
            onDispatch={(input) => onDispatch(entry.id, input)}
          >
            {startButton}
          </DispatchWorkOrderPopover>
        )}
      </PermissionTooltip>
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
