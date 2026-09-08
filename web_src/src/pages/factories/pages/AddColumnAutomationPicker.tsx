import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { Workflow } from "lucide-react";

import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomationCatalogEntry } from "../lib/columnAutomations";

interface AddColumnAutomationPickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (entry: ColumnAutomationCatalogEntry) => void;
  catalog: ColumnAutomationCatalogEntry[];
  takenIds?: readonly string[];
}

export function AddColumnAutomationPicker({
  open,
  onClose,
  onSelect,
  catalog,
  takenIds = [],
}: AddColumnAutomationPickerProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <DialogContent className="gap-0 p-0 sm:max-w-lg" showCloseButton data-testid="add-column-automation-picker">
        <DialogHeader className="border-b border-border px-4 py-3 text-left">
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">
            {COLUMN_AUTOMATIONS_COPY.pickerTitle}
          </DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            {COLUMN_AUTOMATIONS_COPY.pickerDescription}
          </DialogDescription>
        </DialogHeader>

        <ul className="grid grid-cols-2 gap-2 p-3" data-testid="add-column-automation-templates">
          {catalog.map((entry) => {
            const taken = takenIds.includes(entry.id);
            return (
              <li key={entry.id}>
                <button
                  type="button"
                  disabled={taken}
                  onClick={() => {
                    if (taken) {
                      return;
                    }
                    onSelect(entry);
                  }}
                  data-testid={`add-column-automation-template-${entry.id}`}
                  className={cn(
                    "flex h-full min-h-24 w-full flex-col items-start gap-1 rounded-lg border border-border bg-card px-3 py-2.5 text-left shadow-sm transition-colors",
                    taken ? "cursor-not-allowed opacity-60" : "hover:border-foreground/20 hover:bg-accent/40",
                  )}
                >
                  <CatalogGlyph entry={entry} />
                  <span className="text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
                    {entry.name}
                  </span>
                  <span className="workspace-body-text text-muted-foreground">
                    {taken ? COLUMN_AUTOMATIONS_COPY.sourceTaken : entry.description}
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

function CatalogGlyph({ entry }: { entry: ColumnAutomationCatalogEntry }) {
  if (entry.iconSrc) {
    return <img src={entry.iconSrc} alt="" className="size-5 shrink-0" />;
  }
  return <Workflow className="size-5 shrink-0 text-muted-foreground" aria-hidden />;
}
