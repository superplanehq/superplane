import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Forward } from "lucide-react";
import { DispatchWorkOrderPopover } from "../DispatchWorkOrderPopover";
import { OrgUserReference, EmptyOwnerMark } from "../OrgUserReference";
import { OwnerAssignTrigger } from "../WorkOrderAssigneesPopover";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

/** Actions callable from list and table rows, and from the owner on a card. */
export interface WorkOrderRowCallbacks {
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

interface CardOwnerMarkProps {
  entry: WorkOrderListEntry;
  organizationId: string;
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

/**
 * Owner avatar for cards in the top-right corner.
 * Clicking the avatar opens the people list when assignment is allowed.
 */
export function CardOwnerMark({
  entry,
  organizationId,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
}: CardOwnerMarkProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const owner = entry.order.assignees?.[0] as { id?: string; name?: string; avatarUrl?: string } | undefined;
  if (!owner?.id) {
    return (
      <OwnerAssignTrigger
        organizationId={organizationId}
        selectedIds={[]}
        canAssign={canAssign}
        isSaving={isAssigneesSaving}
        onSave={(assigneeIds) => onAssigneesSave(entry.id, assigneeIds)}
        label="Assign owner"
      >
        <span className="inline-flex" data-testid={`work-order-row-assignees-${entry.id}`}>
          <EmptyOwnerMark />
        </span>
      </OwnerAssignTrigger>
    );
  }

  const display = resolveUser(owner.id, owner.name);
  const ownerName = display?.name ?? owner.name;
  if (!ownerName || !display) {
    return null;
  }
  const shown = owner.avatarUrl && !display.avatarUrl ? { ...display, avatarUrl: owner.avatarUrl } : display;

  const mark = (
    <span
      className="inline-flex shrink-0 items-center justify-center"
      data-testid={`work-order-row-assignees-${entry.id}`}
      title={ownerName}
    >
      <OrgUserReference display={shown} size="xs" showName={false} className="rounded-full leading-none" />
    </span>
  );

  return (
    <OwnerAssignTrigger
      organizationId={organizationId}
      selectedIds={[owner.id]}
      canAssign={canAssign}
      isSaving={isAssigneesSaving}
      onSave={(assigneeIds) => onAssigneesSave(entry.id, assigneeIds)}
      label={`Owner: ${ownerName}`}
    >
      {mark}
    </OwnerAssignTrigger>
  );
}

interface AssigneeGroupProps {
  entry: WorkOrderListEntry;
  organizationId: string;
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
  size?: "sm" | "md";
}

/** Single owner avatar. Clicking it opens the people list when assignment is allowed. */
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
  if (!owner?.id) {
    const mark = (
      <span className="inline-flex items-center" data-testid={`work-order-row-assignees-${entry.id}`}>
        <EmptyOwnerMark className="ring-2 ring-background" />
      </span>
    );
    if (!canAssign) {
      return <span className="pointer-events-none inline-flex">{mark}</span>;
    }
    return (
      <OwnerAssignTrigger
        organizationId={organizationId}
        selectedIds={[]}
        canAssign
        isSaving={isAssigneesSaving}
        onSave={(assigneeIds) => onAssigneesSave(entry.id, assigneeIds)}
        label="Assign owner"
      >
        {mark}
      </OwnerAssignTrigger>
    );
  }

  const mark = (
    <span className="inline-flex items-center" data-testid={`work-order-row-assignees-${entry.id}`} title={owner.name}>
      <OrgUserReference
        display={resolveUser(owner.id, owner.name)}
        size={size}
        showName={false}
        className="rounded-full ring-2 ring-background"
      />
    </span>
  );

  if (!canAssign) {
    return <span className="pointer-events-none inline-flex">{mark}</span>;
  }

  return (
    <OwnerAssignTrigger
      organizationId={organizationId}
      selectedIds={[owner.id]}
      canAssign
      isSaving={isAssigneesSaving}
      onSave={(assigneeIds) => onAssigneesSave(entry.id, assigneeIds)}
      label={`Owner: ${owner.name ?? "owner"}`}
    >
      {mark}
    </OwnerAssignTrigger>
  );
}

interface DispatchButtonProps {
  entry: WorkOrderListEntry;
  lines: FactoriesFactoryLine[];
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  /** Only draft/open tasks show the button. */
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
      <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch tasks.">
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
