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
  // The set of assignees as they were when the popover was opened. Used only
  // to keep already-assigned users pinned to the top of the list while it's
  // open, so rows don't jump around as the user toggles checkboxes.
  const [pinnedIds, setPinnedIds] = useState<string[]>(selectedIds);
  const isSaveMode = Boolean(onSave);

  // Note: `draftIds` is intentionally *not* resynced from `selectedIds` on
  // every render where the popover happens to still be open. `selectedIds`
  // is recomputed from the latest server data and can change identity (or
  // even value, via websocket/refetch/window-refocus) while the user is
  // mid-edit; resyncing here would silently discard their pending changes.
  // We only ever want to snapshot the server-confirmed state at the moment
  // the popover transitions from closed to open, which is handled below.
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
      // Nothing actually changed (e.g. the user toggled a checkbox back to
      // its original state) — avoid a pointless request and toast.
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
      <PopoverContent align={align} className="w-72 p-3" sideOffset={8}>
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
