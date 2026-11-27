import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import { SimpleTooltip } from "@/ui/componentSidebar/SimpleTooltip";
import { Camera, Sparkles, Workflow } from "lucide-react";

import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomationCatalogEntry } from "../lib/columnAutomations";

interface AddColumnAutomationPickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (entry: ColumnAutomationCatalogEntry) => void;
  catalog: ColumnAutomationCatalogEntry[];
  takenIds?: readonly string[];
  unavailableReasons?: Readonly<Record<string, string>>;
  title?: string;
}

export function AddColumnAutomationPicker({
  open,
  onClose,
  onSelect,
  catalog,
  takenIds = [],
  unavailableReasons = {},
  title = COLUMN_AUTOMATIONS_COPY.pickerTitle,
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
      <DialogContent className="gap-0 p-0 sm:max-w-2xl" showCloseButton data-testid="add-column-automation-picker">
        <DialogHeader className="border-b border-border px-4 py-3 text-left">
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">{title}</DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            {COLUMN_AUTOMATIONS_COPY.pickerDescription}
          </DialogDescription>
        </DialogHeader>

        <ul
          className="grid max-h-[70vh] grid-cols-1 gap-2 overflow-y-auto p-3 sm:grid-cols-3"
          data-testid="add-column-automation-templates"
        >
          {catalog.map((entry) => {
            const taken = entry.unique && takenIds.includes(entry.id);
            const unavailableReason = unavailableReasons[entry.id];
            const disabled = taken || Boolean(unavailableReason);
            const card = (
              <Button
                type="button"
                variant="ghost"
                disabled={disabled}
                onClick={() => onSelect(entry)}
                data-testid={`add-column-automation-template-${entry.id}`}
                className="h-full min-h-24 w-full flex-col items-start justify-start gap-1 whitespace-normal rounded-lg border border-border bg-card px-3 py-2.5 text-left font-normal shadow-sm hover:border-foreground/20 hover:bg-accent/40 disabled:opacity-60 dark:text-foreground dark:hover:bg-accent/40"
              >
                <CatalogGlyph entry={entry} />
                <span className="text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
                  {entry.name}
                </span>
                <span className="workspace-body-text text-muted-foreground">
                  {taken ? COLUMN_AUTOMATIONS_COPY.sourceTaken : (unavailableReason ?? entry.description)}
                </span>
              </Button>
            );
            return (
              <li key={entry.id}>
                {taken ? (
                  <SimpleTooltip content={<span role="tooltip">{COLUMN_AUTOMATIONS_COPY.sourceTaken}</span>}>
                    <div
                      role="group"
                      aria-label={entry.name}
                      tabIndex={0}
                      className="h-full rounded-lg focus-visible:outline focus-visible:outline-ring"
                    >
                      {card}
                    </div>
                  </SimpleTooltip>
                ) : (
                  card
                )}
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
    return <img src={entry.iconSrc} alt="" className={cn("size-5 shrink-0", logoDarkInvertClass(entry.iconSrc))} />;
  }
  if (entry.kind === "analysis" || entry.kind === "risk-score") {
    return <Sparkles className="size-5 shrink-0 text-muted-foreground" aria-hidden />;
  }
  if (entry.kind === "visual-evidence") {
    return <Camera className="size-5 shrink-0 text-muted-foreground" aria-hidden />;
  }
  return <Workflow className="size-5 shrink-0 text-muted-foreground" aria-hidden />;
}
