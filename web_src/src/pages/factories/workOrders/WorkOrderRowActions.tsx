import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Forward } from "lucide-react";
import { DispatchWorkOrderPopover } from "../DispatchWorkOrderPopover";
import { OrgUserReference } from "../OrgUserReference";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

/** Actions callable from list and table rows. Cards do not change the owner. */
export interface WorkOrderRowCallbacks {
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}

interface CardOwnerMarkProps {
  entry: WorkOrderListEntry;
  organizationId: string;
}

/**
 * Display-only owner avatar for cards. The owner cannot be changed here.
 */
export function CardOwnerMark({ entry, organizationId }: CardOwnerMarkProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  if (entry.displayStatus === "draft") {
    return null;
  }

  const owner = entry.order.assignees?.[0];
  if (!owner) {
    return null;
  }

  return (
    <span
      className="inline-flex size-5 shrink-0 items-center justify-center"
      data-testid={`work-order-row-assignees-${entry.id}`}
      title={owner.name}
    >
      <OrgUserReference
        display={resolveUser(owner.id, owner.name)}
        size="xs"
        showName={false}
        className="rounded-full leading-none"
      />
    </span>
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

/**
 * Single owner avatar. The owner cannot be changed here.
 */
export function AssigneeGroup({ entry, organizationId, size = "sm" }: AssigneeGroupProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const owner = entry.order.assignees?.[0];
  if (!owner) {
    return null;
  }

  return (
    <span
      className="pointer-events-none inline-flex items-center"
      data-testid={`work-order-row-assignees-${entry.id}`}
      title={owner.name}
    >
      <OrgUserReference
        display={resolveUser(owner.id, owner.name)}
        size={size}
        showName={false}
        className="rounded-full ring-2 ring-background"
      />
    </span>
  );
}

/** Line to start on: the preferred line when it exists, else the only line. */
function resolveStartLineName(lines: FactoriesFactoryLine[], preferredLineName?: string): string | undefined {
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
 * Persistent Start control on a draft card. One click sends the task
 * to the preferred line, or opens the line picker when more than one line
 * exists.
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
    <LoadingButton
      type="button"
      size="xs"
      disabled={disabled}
      loading={isDispatching}
      loadingText="Starting..."
      data-testid={`work-order-card-start-${entry.id}`}
      onClick={lineName ? () => void onDispatch(entry.id, { lineName }) : undefined}
    >
      Start
    </LoadingButton>
  );

  return (
    <div className="pointer-events-auto" onClick={(event) => event.stopPropagation()}>
      <PermissionTooltip allowed={canDispatch} message="You don't have permission to start this task.">
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
