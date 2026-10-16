import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Plus } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import {
  addIntakeLabel,
  INTAKE_SETTINGS_COPY,
  toggleIntakeLabel,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

/** The GitHub label SuperPlane uses as the primary intake label. */
export const GITHUB_INTAKE_MAIN_LABEL = "superplane";

export function GitHubIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  labelOptions = [],
  labelOptionsLoading = false,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  /** Labels that exist in the connected repository. */
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
}) {
  if (sourceId !== "github-issues") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  function toggleMainLabel() {
    onSettingsChange((current) => {
      const labels = toggleIntakeLabel(current.labels, GITHUB_INTAKE_MAIN_LABEL);
      return { ...current, labels, filterByLabel: labels.length > 0, labelFilterMode: "include" };
    });
  }

  function setOtherLabels(otherLabels: string[]) {
    onSettingsChange((current) => {
      const labels = labelsForSelection(current.labels.includes(GITHUB_INTAKE_MAIN_LABEL), otherLabels);
      return { ...current, labels, filterByLabel: labels.length > 0, labelFilterMode: "include" };
    });
  }

  const mainLabelSelected = settings.labels.includes(GITHUB_INTAKE_MAIN_LABEL);
  const otherLabels = settings.labels.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL);
  const otherLabelOptions = labelOptions.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL);

  return (
    <div className="flex flex-col gap-6">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-4">
          <IntakeSettingsCheckbox
            title={INTAKE_SETTINGS_COPY.authorsWithAccess}
            checked={settings.authorsWithAccess}
            onChange={() => update("authorsWithAccess", !settings.authorsWithAccess)}
          />
          <div className="flex flex-col gap-2">
            <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.mainLabel}</p>
            <IntakeSettingsCheckbox
              title={GITHUB_INTAKE_MAIN_LABEL}
              checked={mainLabelSelected}
              onChange={() => toggleMainLabel()}
            />
          </div>
          <div className="flex flex-col gap-2">
            <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.otherLabels}</p>
            <IntakeLabelField
              labels={otherLabels}
              options={otherLabelOptions}
              loading={labelOptionsLoading}
              onChange={(labels) => setOtherLabels(labels)}
            />
          </div>
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{INTAKE_SETTINGS_COPY.intakeSection}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <IntakeSettingsCheckbox
            title={INTAKE_SETTINGS_COPY.reopenedIssues}
            checked={settings.reopenedIssues}
            onChange={() => update("reopenedIssues", !settings.reopenedIssues)}
          />
        </div>
      </fieldset>
    </div>
  );
}

function labelsForSelection(mainLabelSelected: boolean, otherLabels: string[]): string[] {
  const includeMain = mainLabelSelected || otherLabels.includes(GITHUB_INTAKE_MAIN_LABEL);
  const others = otherLabels.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL);
  if (!includeMain) {
    return others;
  }
  return [GITHUB_INTAKE_MAIN_LABEL, ...others];
}

interface IntakeLabelFieldProps {
  labels: string[];
  options: string[];
  loading: boolean;
  onChange: (labels: string[]) => void;
}

function IntakeLabelField({ labels, options, loading, onChange }: IntakeLabelFieldProps) {
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const choices = [...options, ...labels.filter((label) => !options.includes(label))];

  useEffect(() => {
    if (typing) {
      inputRef.current?.focus();
    }
  }, [typing]);

  function add(label: string) {
    onChange(addIntakeLabel(labels, label));
  }

  function close() {
    setDraft("");
    setTyping(false);
  }

  function addDraft() {
    add(draft);
    close();
  }

  return (
    <div className="flex flex-col gap-2" data-testid="intake-label-options">
      {loading ? <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsLoading}</p> : null}
      {!loading && choices.length === 0 ? (
        <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsEmpty}</p>
      ) : null}
      {choices.length > 0 ? (
        <ul className="flex flex-wrap items-center gap-1.5">
          {choices.map((label) => {
            const checked = labels.includes(label);
            return (
              <li key={label}>
                <label
                  className={cn(
                    "inline-flex max-w-full cursor-pointer items-center gap-2 rounded-md border px-2 py-1 text-[13px]",
                    checked
                      ? "border-foreground/20 bg-accent/50 text-foreground"
                      : "border-border bg-card text-muted-foreground hover:border-foreground/15",
                  )}
                >
                  <Checkbox
                    checked={checked}
                    onChange={() => onChange(toggleIntakeLabel(labels, label))}
                    aria-label={label}
                  />
                  <span className="min-w-0 truncate">{label}</span>
                </label>
              </li>
            );
          })}
        </ul>
      ) : null}
      {typing ? (
        <div className="flex items-center gap-2">
          <Input
            ref={inputRef}
            value={draft}
            aria-label={INTAKE_SETTINGS_COPY.labelInput}
            placeholder={INTAKE_SETTINGS_COPY.labelPlaceholder}
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
            data-testid="intake-label-input"
          />
          <Button
            type="button"
            variant="outline"
            className="shrink-0"
            disabled={draft.trim().length === 0}
            onClick={() => addDraft()}
            data-testid="intake-label-add"
          >
            {INTAKE_SETTINGS_COPY.labelAdd}
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="shrink-0"
            onClick={() => close()}
            data-testid="intake-label-cancel"
          >
            {INTAKE_SETTINGS_COPY.labelCancel}
          </Button>
        </div>
      ) : (
        <div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setTyping(true)}
            data-testid="intake-label-new"
          >
            <Plus className="size-3.5" aria-hidden />
            {INTAKE_SETTINGS_COPY.labelNew}
          </Button>
        </div>
      )}
    </div>
  );
}

function IntakeSettingsCheckbox({
  title,
  checked,
  onChange,
}: {
  title: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox checked={checked} onChange={onChange} aria-label={title} />
      <span className="min-w-0 text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
    </label>
  );
}
