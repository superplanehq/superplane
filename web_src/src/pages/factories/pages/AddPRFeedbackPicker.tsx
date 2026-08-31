import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

import {
  PR_FEEDBACK_SETTINGS_COPY,
  PR_FEEDBACK_SOURCES,
  type PRFeedbackSource,
  type PRFeedbackSourceId,
} from "./prFeedbackSettingsModel";

interface AddPRFeedbackPickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (source: PRFeedbackSource) => void;
  takenSourceIds?: readonly PRFeedbackSourceId[];
}

export function AddPRFeedbackPicker({
  open,
  onClose,
  onSelect,
  takenSourceIds = [],
}: AddPRFeedbackPickerProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <DialogContent className="gap-0 p-0 sm:max-w-lg" showCloseButton data-testid="add-pr-feedback-picker">
        <DialogHeader className="border-b border-border px-4 py-3 text-left">
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">
            {PR_FEEDBACK_SETTINGS_COPY.pickerTitle}
          </DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            {PR_FEEDBACK_SETTINGS_COPY.pickerDescription}
          </DialogDescription>
        </DialogHeader>

        <ul className="grid grid-cols-2 gap-2 p-3" data-testid="add-pr-feedback-templates">
          {PR_FEEDBACK_SOURCES.map((source) => {
            const taken = takenSourceIds.includes(source.id);
            return (
              <li key={source.id}>
                <button
                  type="button"
                  disabled={taken}
                  onClick={() => {
                    if (taken) {
                      return;
                    }
                    onSelect(source);
                  }}
                  data-testid={`add-pr-feedback-template-${source.id}`}
                  className={cn(
                    "flex h-full min-h-24 w-full flex-col items-start gap-1 rounded-lg border border-border bg-card px-3 py-2.5 text-left shadow-sm transition-colors",
                    taken
                      ? "cursor-not-allowed opacity-60"
                      : "hover:border-foreground/20 hover:bg-accent/40",
                  )}
                >
                  <img src={source.iconSrc} alt="" className="size-5 shrink-0" />
                  <span className="text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
                    {source.name}
                  </span>
                  <span className="workspace-body-text text-muted-foreground">
                    {taken ? PR_FEEDBACK_SETTINGS_COPY.sourceTaken : source.description}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </DialogContent>
    </Dialog>
  );
}
