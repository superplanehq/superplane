import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useState, type ReactNode } from "react";
import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

interface WorkOrderAssigneesPopoverProps {
  organizationId: string;
  factoryId?: string;
  orderId?: string;
  selectedIds: string[];
  align?: "start" | "center" | "end";
  disabled?: boolean;
  canEdit?: boolean;
  isSaving?: boolean;
  onChange?: (assigneeIds: string[]) => void;
  onSave?: (assigneeIds: string[]) => Promise<void>;
  children: ReactNode;
}

function haveSameIds(left: string[], right: string[]): boolean {
  if (left.length !== right.length) {
    return false;
  }

  const sortedLeft = [...left].sort();
  const sortedRight = [...right].sort();
  return sortedLeft.every((id, index) => id === sortedRight[index]);
}

export function WorkOrderAssigneesPopover({
  organizationId,
  factoryId,
  orderId,
  selectedIds,
  align = "end",
  disabled = false,
  canEdit = true,
  isSaving = false,
  onChange,
  onSave,
  children,
}: WorkOrderAssigneesPopoverProps) {
  const [open, setOpen] = useState(false);

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }

    setOpen(nextOpen);
  };

  const handleChange = async (nextIds: string[]) => {
    if (haveSameIds(nextIds, selectedIds)) {
      setOpen(false);
      return;
    }

    if (onSave) {
      try {
        await onSave(nextIds);
        setOpen(false);
      } catch {
        // Caller shows error toast; keep popover open for retry.
      }
      return;
    }

    onChange?.(nextIds);
    setOpen(false);
  };

  const pickerDisabled = disabled || isSaving || !canEdit;

  return (
    <Popover open={open} onOpenChange={handleOpenChange} modal={false}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="z-[70] w-72 overflow-hidden p-0" sideOffset={8}>
        <WorkOrderAssigneePicker
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          selectedIds={selectedIds}
          onChange={(nextIds) => void handleChange(nextIds)}
          disabled={pickerDisabled}
        />
      </PopoverContent>
    </Popover>
  );
}

/** Click target for the current owner. Opens the people list when assignment is allowed. */
export function OwnerAssignTrigger({
  organizationId,
  factoryId,
  orderId,
  selectedIds,
  canAssign,
  isSaving = false,
  onSave,
  align = "end",
  label,
  testId,
  children,
}: {
  organizationId: string;
  factoryId?: string;
  orderId?: string;
  selectedIds: string[];
  canAssign: boolean;
  isSaving?: boolean;
  onSave: (assigneeIds: string[]) => Promise<void>;
  align?: "start" | "center" | "end";
  label: string;
  testId?: string;
  children: ReactNode;
}) {
  if (!canAssign) {
    return children;
  }

  return (
    <span className="pointer-events-auto inline-flex min-w-0 max-w-full" onClick={(event) => event.stopPropagation()}>
      <WorkOrderAssigneesPopover
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        selectedIds={selectedIds}
        align={align}
        canEdit
        isSaving={isSaving}
        onSave={onSave}
      >
        <button
          type="button"
          className="inline-flex h-5 min-w-0 max-w-full items-center rounded-sm p-0 text-left hover:bg-muted/60 disabled:opacity-60"
          aria-label={label}
          data-testid={testId}
          disabled={isSaving}
        >
          {children}
        </button>
      </WorkOrderAssigneesPopover>
    </span>
  );
}
