import type { CanvasesCanvasVersion } from "@/api-client";
import { Button as UIButton } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { StartEditingDraftPicker } from "./StartEditingDraftPicker";

export type StartEditingDropdownProps = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  drafts: CanvasesCanvasVersion[];
  defaultDraft: CanvasesCanvasVersion | null;
  disabled?: boolean;
  disabledTooltip?: string;
  isSubmitting?: boolean;
  onContinueDraft: (branchName: string) => void;
  onCreateDraft: () => void;
};

export function StartEditingDropdown({
  open,
  onOpenChange,
  drafts,
  defaultDraft,
  disabled,
  isSubmitting,
  onContinueDraft,
  onCreateDraft,
}: StartEditingDropdownProps) {
  const handleOpenChange = (nextOpen: boolean) => {
    onOpenChange?.(nextOpen);
  };

  if (drafts.length === 0) {
    return (
      <UIButton
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || isSubmitting}
        data-testid="canvas-edit-button"
        onClick={onCreateDraft}
      >
        Edit
      </UIButton>
    );
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <UIButton type="button" variant="outline" size="sm" disabled={disabled} data-testid="canvas-edit-button">
          Edit
        </UIButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72 px-2" data-testid="start-editing-menu">
        <StartEditingDraftPicker
          drafts={drafts}
          defaultDraft={defaultDraft}
          isSubmitting={isSubmitting}
          onContinueDraft={onContinueDraft}
          onCreateDraft={onCreateDraft}
          onClose={() => handleOpenChange(false)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
