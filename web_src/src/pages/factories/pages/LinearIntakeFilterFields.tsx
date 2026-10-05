import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { Plus } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { addIntakeLabel, toggleIntakeLabel, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";
import { LinearProjectPicker } from "./LinearProjectPicker";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function LinearIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
}) {
  const projectsQuery = useIntegrationResources(organizationId ?? "", integrationId ?? "", "project", undefined, {
    enabled: sourceId === "linear-issues" && Boolean(organizationId && integrationId),
  });

  if (sourceId !== "linear-issues") {
    return null;
  }

  const toggleProject = (id: string) => {
    onSettingsChange((current) => ({
      ...current,
      linearProjectIds: current.linearProjectIds.includes(id)
        ? current.linearProjectIds.filter((entry) => entry !== id)
        : [...current.linearProjectIds, id],
    }));
  };

  return (
    <div className="flex flex-col gap-6" data-testid="linear-intake-filters">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.projectsLabel}</legend>
        <div className="mt-2">
          <LinearProjectPicker
            projects={projectsQuery.data ?? []}
            selectedIds={settings.linearProjectIds}
            loading={projectsQuery.isLoading}
            error={projectsQuery.isError}
            onToggle={toggleProject}
            onRetry={() => void projectsQuery.refetch()}
          />
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.labelsLabel}</legend>
        <p className="workspace-body-text mt-1 text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsHelper}</p>
        <div className="mt-2">
          <LinearLabelField
            labels={settings.linearLabels}
            onChange={(linearLabels) => onSettingsChange((current) => ({ ...current, linearLabels }))}
          />
        </div>
      </fieldset>
    </div>
  );
}

export function LinearLabelField({ labels, onChange }: { labels: string[]; onChange: (labels: string[]) => void }) {
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (typing) {
      inputRef.current?.focus();
    }
  }, [typing]);

  function close() {
    setDraft("");
    setTyping(false);
  }

  function addDraft() {
    onChange(addIntakeLabel(labels, draft));
    close();
  }

  return (
    <div className="flex flex-col gap-2" data-testid="linear-intake-label-options">
      {labels.length > 0 ? (
        <ul className="flex flex-wrap items-center gap-1.5">
          {labels.map((label) => (
            <li key={label}>
              <label
                className={cn(
                  "inline-flex max-w-full cursor-pointer items-center gap-2 rounded-md border px-2 py-1 text-[13px]",
                  "border-foreground/20 bg-accent/50 text-foreground",
                )}
              >
                <Checkbox checked onChange={() => onChange(toggleIntakeLabel(labels, label))} aria-label={label} />
                <span className="min-w-0 truncate">{label}</span>
              </label>
            </li>
          ))}
        </ul>
      ) : null}
      {typing ? (
        <div className="flex items-center gap-2">
          <Input
            ref={inputRef}
            value={draft}
            aria-label={LINEAR_INTAKE_SETUP_COPY.labelInput}
            placeholder={LINEAR_INTAKE_SETUP_COPY.labelPlaceholder}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                event.preventDefault();
                close();
                return;
              }
              if (event.key !== "Enter") {
                return;
              }
              event.preventDefault();
              addDraft();
            }}
            data-testid="linear-intake-label-input"
          />
          <Button
            type="button"
            variant="outline"
            className="shrink-0"
            disabled={draft.trim().length === 0}
            onClick={() => addDraft()}
          >
            {LINEAR_INTAKE_SETUP_COPY.labelAdd}
          </Button>
          <Button type="button" variant="ghost" className="shrink-0" onClick={() => close()}>
            {LINEAR_INTAKE_SETUP_COPY.labelCancel}
          </Button>
        </div>
      ) : (
        <div>
          <Button type="button" variant="outline" size="sm" onClick={() => setTyping(true)}>
            <Plus className="size-3.5" aria-hidden />
            {LINEAR_INTAKE_SETUP_COPY.labelNew}
          </Button>
        </div>
      )}
    </div>
  );
}
