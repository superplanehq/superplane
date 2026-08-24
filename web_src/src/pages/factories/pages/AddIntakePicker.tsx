import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { filterAddIntakeTemplates, type AddIntakeTemplate } from "./lineIntakeModel";

interface AddIntakePickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (template: AddIntakeTemplate) => void;
}

/**
 * Lightweight picker: search, then choose one intake template box.
 */
export function AddIntakePicker({ open, onClose, onSelect }: AddIntakePickerProps) {
  const [query, setQuery] = useState("");
  const templates = useMemo(() => filterAddIntakeTemplates(query), [query]);

  useEffect(() => {
    if (open) {
      setQuery("");
    }
  }, [open]);

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
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">Add intake</DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            Choose a template for a new intake automation.
          </DialogDescription>
        </DialogHeader>

        <div className="border-b border-border px-4 py-3">
          <div className="relative">
            <Search
              className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
              aria-hidden
            />
            <Input
              id="add-intake-search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search intakes"
              aria-label="Search intakes"
              className="h-9 pl-8 text-[13px] shadow-none"
              data-testid="add-intake-search"
              autoFocus
            />
          </div>
        </div>

        <ul
          className="grid max-h-[min(24rem,50vh)] grid-cols-2 gap-2 overflow-y-auto p-3 [scrollbar-width:thin]"
          data-testid="add-intake-templates"
        >
          {templates.length === 0 ? (
            <li className="col-span-2 px-2 py-8 text-center">
              <p className="workspace-body-text text-muted-foreground">No intakes match this search.</p>
            </li>
          ) : (
            templates.map((template) => (
              <li key={template.id}>
                <button
                  type="button"
                  onClick={() => onSelect(template)}
                  data-testid={`add-intake-template-${template.id}`}
                  className="flex h-full min-h-24 w-full flex-col items-start gap-1 rounded-lg border border-border bg-card px-3 py-2.5 text-left shadow-sm transition-colors hover:border-foreground/20 hover:bg-accent/40"
                >
                  <TemplateGlyph template={template} />
                  <span className="text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
                    {template.name}
                  </span>
                  <span className="workspace-body-text text-muted-foreground">{template.description}</span>
                </button>
              </li>
            ))
          )}
        </ul>
      </DialogContent>
    </Dialog>
  );
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
