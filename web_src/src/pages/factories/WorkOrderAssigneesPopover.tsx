import { LoadingButton } from "@/components/ui/loading-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useEffect, useState, type ReactNode } from "react";
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
  const isSaveMode = Boolean(onSave);

  useEffect(() => {
    if (open) {
      setDraftIds(selectedIds);
    }
  }, [open, selectedIds]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
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
