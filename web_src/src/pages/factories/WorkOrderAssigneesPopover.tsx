import { LoadingButton } from "@/components/ui/loading-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useState, type ReactNode } from "react";
import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

interface WorkOrderAssigneesPopoverProps {
  organizationId: string;
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
  const [draftIds, setDraftIds] = useState<string[]>(selectedIds);
  // Snapshot of the confirmed selection, used to keep assigned users pinned
  // to the top of the list while the popover is open.
  const [pinnedIds, setPinnedIds] = useState<string[]>(selectedIds);
  const isSaveMode = Boolean(onSave);

  // `draftIds` is only synced from `selectedIds` when the popover opens, not
  // on every render, so it doesn't discard pending edits.
  const handleOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }

    if (nextOpen && !open) {
      setDraftIds(selectedIds);
      setPinnedIds(selectedIds);
    }

    setOpen(nextOpen);

    if (!nextOpen && isSaveMode) {
      setDraftIds(selectedIds);
    }
  };

  const handleChange = (nextIds: string[]) => {
    if (isSaveMode) {
      setDraftIds(nextIds);
      return;
    }

    onChange?.(nextIds);
  };

  const handleSave = async () => {
    if (!onSave) {
      return;
    }

    if (haveSameIds(draftIds, selectedIds)) {
      // No actual change; skip the request.
      setOpen(false);
      return;
    }

    try {
      await onSave(draftIds);
      setOpen(false);
    } catch {
      // Caller shows error toast; keep popover open for retry.
    }
  };

  const activeIds = isSaveMode ? draftIds : selectedIds;
  const pickerDisabled = disabled || isSaving || !canEdit;

  return (
    <Popover open={open} onOpenChange={handleOpenChange} modal={false}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="z-[70] w-72 p-3" sideOffset={8}>
        <div className="space-y-3">
          <WorkOrderAssigneePicker
            organizationId={organizationId}
            selectedIds={activeIds}
            pinnedIds={isSaveMode ? pinnedIds : selectedIds}
            onChange={handleChange}
            disabled={pickerDisabled}
            variant="popover"
          />

          {isSaveMode ? (
            <LoadingButton
              type="button"
              onClick={() => void handleSave()}
              disabled={pickerDisabled}
              loading={isSaving}
              loadingText="Saving..."
              className="w-full"
              data-testid="work-order-save-assignees"
            >
              Save
            </LoadingButton>
          ) : null}
        </div>
      </PopoverContent>
    </Popover>
  );
}
