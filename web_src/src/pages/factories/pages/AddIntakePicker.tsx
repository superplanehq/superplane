import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

import { ADD_INTAKE_COPY, ADD_INTAKE_TEMPLATES, type AddIntakeTemplate } from "./lineIntakeModel";

interface AddIntakePickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (template: AddIntakeTemplate) => void;
  /** Templates offered by the picker. Defaults to the full template catalog. */
  templates?: AddIntakeTemplate[];
  /** Source ids that already have an intake. Those cards stay disabled. */
  takenSourceIds?: readonly string[];
}

export function AddIntakePicker({
  open,
  onClose,
  onSelect,
  templates = ADD_INTAKE_TEMPLATES,
  takenSourceIds = [],
}: AddIntakePickerProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose();
        }
      }}
    >
      <DialogContent className="gap-0 p-0 sm:max-w-lg" showCloseButton data-testid="add-intake-picker">
        <DialogHeader className="border-b border-border px-4 py-3 text-left">
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">
            {ADD_INTAKE_COPY.pickerTitle}
          </DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            {ADD_INTAKE_COPY.pickerDescription}
          </DialogDescription>
        </DialogHeader>

        <ul className="grid grid-cols-2 gap-2 p-3" data-testid="add-intake-templates">
          {templates.map((template) => {
            const taken = takenSourceIds.includes(template.id);
            const unavailable = Boolean(template.soon) || taken;
            return (
              <li key={template.id}>
                <button
                  type="button"
                  disabled={unavailable}
                  onClick={() => {
                    if (unavailable) {
                      return;
                    }
                    onSelect(template);
                  }}
                  data-testid={`add-intake-template-${template.id}`}
                  className={cn(
                    "flex h-full min-h-24 w-full flex-col items-start gap-1 rounded-lg border border-border bg-card px-3 py-2.5 text-left shadow-sm transition-colors",
                    unavailable ? "cursor-not-allowed opacity-60" : "hover:border-foreground/20 hover:bg-accent/40",
                  )}
                >
                  <TemplateGlyph template={template} />
                  <span className="text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
                    {template.name}
                  </span>
                  <span className="workspace-body-text text-muted-foreground">
                    {intakeTemplateStatus(template, taken)}
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

function intakeTemplateStatus(template: AddIntakeTemplate, taken: boolean): string {
  if (taken) {
    return ADD_INTAKE_COPY.sourceTaken;
  }
  if (template.soon) {
    return ADD_INTAKE_COPY.comingSoon;
  }
  return template.description;
}

function TemplateGlyph({ template }: { template: AddIntakeTemplate }) {
  if (template.iconSrc) {
    return <img src={template.iconSrc} alt="" className="size-5 shrink-0" />;
  }
  return (
    <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted text-[11px] font-medium text-muted-foreground">
      {template.name.charAt(0)}
    </span>
  );
}
