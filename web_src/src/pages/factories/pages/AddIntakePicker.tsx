import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import { Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { ADD_INTAKE_COPY, type AddIntakeTemplate } from "./lineIntakeModel";

interface AddIntakePickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (template: AddIntakeTemplate) => void;
  /** Templates offered by the picker, from the organization's intake catalog. */
  templates: AddIntakeTemplate[];
  loading?: boolean;
  loadFailed?: boolean;
  /** Source ids that already have an intake. Those cards stay disabled. */
  takenSourceIds?: readonly string[];
}

function filterTemplates(templates: AddIntakeTemplate[], query: string): AddIntakeTemplate[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return templates;
  }
  return templates.filter((template) => {
    const haystack = `${template.name} ${template.description}`.toLowerCase();
    return haystack.includes(needle);
  });
}

export function AddIntakePicker({
  open,
  onClose,
  onSelect,
  templates: availableTemplates,
  loading = false,
  loadFailed = false,
  takenSourceIds = [],
}: AddIntakePickerProps) {
  const [query, setQuery] = useState("");
  const templates = useMemo(() => filterTemplates(availableTemplates, query), [availableTemplates, query]);

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
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">
            {ADD_INTAKE_COPY.pickerTitle}
          </DialogTitle>
          <DialogDescription className="workspace-body-text text-muted-foreground">
            {ADD_INTAKE_COPY.pickerDescription}
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
          {loading || loadFailed || templates.length === 0 ? (
            <li className="col-span-2 px-2 py-8 text-center">
              <p className="workspace-body-text text-muted-foreground">{pickerEmptyMessage(loading, loadFailed)}</p>
            </li>
          ) : (
            templates.map((template) => (
              <TemplateCard
                key={template.id}
                template={template}
                taken={takenSourceIds.includes(template.id)}
                onSelect={onSelect}
              />
            ))
          )}
        </ul>
      </DialogContent>
    </Dialog>
  );
}

function pickerEmptyMessage(loading: boolean, loadFailed: boolean): string {
  if (loading) {
    return ADD_INTAKE_COPY.loading;
  }
  if (loadFailed) {
    return ADD_INTAKE_COPY.loadFailed;
  }
  return "No intakes match this search.";
}

interface TemplateCardProps {
  template: AddIntakeTemplate;
  taken: boolean;
  /** Omit to render the card read-only. */
  onSelect?: (template: AddIntakeTemplate) => void;
}

export function TemplateCard({ template, taken, onSelect }: TemplateCardProps) {
  const unavailable = Boolean(template.soon) || taken;
  const readOnly = !onSelect;
  return (
    <li>
      <button
        type="button"
        disabled={unavailable}
        tabIndex={readOnly ? -1 : undefined}
        onClick={() => {
          if (unavailable || !onSelect) {
            return;
          }
          onSelect(template);
        }}
        data-testid={`add-intake-template-${template.id}`}
        className={cn(
          "flex h-full min-h-24 w-full flex-col items-start gap-1 rounded-lg border border-border bg-card px-3 py-2.5 text-left shadow-sm transition-colors",
          unavailable && "cursor-not-allowed opacity-60",
          !unavailable && !readOnly && "hover:border-foreground/20 hover:bg-accent/40",
          readOnly && "pointer-events-none",
        )}
      >
        <TemplateGlyph template={template} />
        <span className="flex items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em] leading-5 text-foreground">
          {template.name}
          {template.beta ? (
            <span
              data-testid="add-intake-beta-badge"
              className="rounded-full border border-amber-300/70 bg-amber-50 px-1.5 text-[10px] font-medium leading-4 text-amber-800 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-200"
            >
              {ADD_INTAKE_COPY.beta}
            </span>
          ) : null}
        </span>
        <span className="workspace-body-text text-muted-foreground">{intakeTemplateStatus(template, taken)}</span>
      </button>
    </li>
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
    return (
      <img src={template.iconSrc} alt="" className={cn("size-5 shrink-0", logoDarkInvertClass(template.iconSrc))} />
    );
  }
  return (
    <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted text-[11px] font-medium text-muted-foreground">
      {template.name.charAt(0)}
    </span>
  );
}
