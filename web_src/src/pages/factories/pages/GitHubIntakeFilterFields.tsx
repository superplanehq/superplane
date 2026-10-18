import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Plus, X } from "lucide-react";
import { useEffect, useState, type Dispatch, type SetStateAction } from "react";

import { addIntakeLabel, INTAKE_SETTINGS_COPY, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
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

  function setLabels(labels: string[]) {
    onSettingsChange((current) => ({
      ...current,
      labels,
      filterByLabel: labels.length > 0 ? true : current.filterByLabel,
      labelFilterMode: "include",
    }));
  }

  function toggleFilterByLabel() {
    onSettingsChange((current) => {
      const filterByLabel = !current.filterByLabel;
      if (filterByLabel) {
        return { ...current, filterByLabel, labelFilterMode: "include" };
      }
      return { ...current, filterByLabel, labels: [], labelFilterMode: "include" };
    });
  }

  return (
    <div className="flex flex-col gap-6">
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
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <IntakeSettingsCheckbox
              title={INTAKE_SETTINGS_COPY.filterByLabel}
              checked={settings.filterByLabel}
              onChange={() => toggleFilterByLabel()}
            />
            {settings.filterByLabel || settings.labels.length > 0 ? (
              <div className="flex flex-col gap-2">
                <IntakeLabelField
                  labels={settings.labels}
                  options={labelOptions}
                  loading={labelOptionsLoading}
                  onChange={(labels) => setLabels(labels)}
                />
                {settings.labels.length > 0 ? (
                  <p className="ml-6 text-[12px] leading-5 text-muted-foreground">
                    {INTAKE_SETTINGS_COPY.labelAddedHelper}
                  </p>
                ) : null}
              </div>
            ) : null}
          </div>
          <IntakeSettingsCheckbox
            title={INTAKE_SETTINGS_COPY.authorsWithAccess}
            checked={settings.authorsWithAccess}
            onChange={() => update("authorsWithAccess", !settings.authorsWithAccess)}
          />
        </div>
      </fieldset>
    </div>
  );
}

interface IntakeLabelFieldProps {
  labels: string[];
  options: string[];
  loading: boolean;
  onChange: (labels: string[]) => void;
}

function IntakeLabelField({ labels, options, loading, onChange }: IntakeLabelFieldProps) {
  const [query, setQuery] = useState("");
  const [addingLabel, setAddingLabel] = useState(false);
  const view = intakeLabelView(labels, options, query, loading);
  const showSearch = !view.allSelected || addingLabel;

  useEffect(() => {
    if (view.allSelected && !addingLabel) {
      setQuery("");
    }
  }, [view.allSelected, addingLabel]);

  function addQuery() {
    onChange(labelsAfterQuery(labels, view.choices, query));
    setQuery("");
    setAddingLabel(false);
  }

  return (
    <div className="mb-1 ml-6 flex flex-col gap-2" data-testid="intake-label-options">
      <IntakeSelectedLabels
        labels={view.selected}
        onRemove={(label) => onChange(labels.filter((entry) => entry !== label))}
      />
      {showSearch ? (
        <IntakeLabelQuery
          query={query}
          placeholder={
            view.allSelected ? INTAKE_SETTINGS_COPY.labelNamePlaceholder : INTAKE_SETTINGS_COPY.labelPlaceholder
          }
          onQueryChange={setQuery}
          onCancel={() => {
            setQuery("");
            setAddingLabel(false);
          }}
          onSubmit={() => addQuery()}
        />
      ) : (
        <IntakeLabelsAllSelected onAdd={() => setAddingLabel(true)} />
      )}
      {loading ? <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsLoading}</p> : null}
      {view.available.length > 0 ? (
        <ul className="flex max-h-44 flex-col gap-0.5 overflow-y-auto rounded-lg border border-border bg-card p-1">
          {view.available.map((label) => (
            <IntakeLabelOption key={label} label={label} onSelect={() => onChange([...labels, label])} />
          ))}
        </ul>
      ) : null}
      {view.showNoMatch ? (
        <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsNoMatch}</p>
      ) : null}
      {view.canCreate ? <IntakeCreateLabel label={view.trimmedQuery} onCreate={() => addQuery()} /> : null}
    </div>
  );
}

interface IntakeLabelView {
  choices: string[];
  selected: string[];
  available: string[];
  allSelected: boolean;
  canCreate: boolean;
  showNoMatch: boolean;
  trimmedQuery: string;
}

function intakeLabelView(labels: string[], options: string[], query: string, loading: boolean): IntakeLabelView {
  const choices = intakeLabelChoices(labels, options);
  const trimmedQuery = query.trim();
  const available = loading
    ? []
    : choices.filter((label) => !labels.includes(label) && label.toLowerCase().includes(trimmedQuery.toLowerCase()));
  const allSelected = !loading && choices.length > 0 && choices.every((label) => labels.includes(label));
  const canCreate =
    trimmedQuery.length > 0 && !choices.some((label) => label.toLowerCase() === trimmedQuery.toLowerCase());

  return {
    choices,
    selected: intakeSelectedLabels(labels),
    available,
    allSelected,
    canCreate,
    showNoMatch: !loading && !allSelected && trimmedQuery.length > 0 && available.length === 0 && !canCreate,
    trimmedQuery,
  };
}

function labelsAfterQuery(labels: string[], choices: string[], query: string): string[] {
  const trimmedQuery = query.trim();
  if (trimmedQuery.length === 0) {
    return labels;
  }
  const exactChoice = choices.find((label) => label.toLowerCase() === trimmedQuery.toLowerCase());
  if (exactChoice) {
    return labels.includes(exactChoice) ? labels : [...labels, exactChoice];
  }
  return addIntakeLabel(labels, trimmedQuery);
}

function intakeLabelChoices(labels: string[], options: string[]): string[] {
  return [
    GITHUB_INTAKE_MAIN_LABEL,
    ...options.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL),
    ...labels.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL && !options.includes(label)),
  ];
}

function intakeSelectedLabels(labels: string[]): string[] {
  return [
    ...(labels.includes(GITHUB_INTAKE_MAIN_LABEL) ? [GITHUB_INTAKE_MAIN_LABEL] : []),
    ...labels.filter((label) => label !== GITHUB_INTAKE_MAIN_LABEL),
  ];
}

function IntakeSelectedLabels({ labels, onRemove }: { labels: string[]; onRemove: (label: string) => void }) {
  if (labels.length === 0) {
    return null;
  }

  return (
    <ul className="flex flex-wrap items-center gap-1.5" aria-label="Selected labels">
      {labels.map((label) => (
        <IntakeSelectedLabel key={label} label={label} onRemove={onRemove} />
      ))}
    </ul>
  );
}

function IntakeSelectedLabel({ label, onRemove }: { label: string; onRemove: (label: string) => void }) {
  return (
    <li className="inline-flex max-w-full items-center gap-1 rounded-md border border-foreground/20 bg-accent/50 py-1 pr-1 pl-2 text-[13px] text-foreground">
      <span className="min-w-0 truncate">{label}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-5"
        aria-label={`Remove ${label}`}
        onClick={() => onRemove(label)}
      >
        <X className="size-3.5" aria-hidden />
      </Button>
    </li>
  );
}

function IntakeLabelQuery({
  query,
  placeholder,
  onQueryChange,
  onCancel,
  onSubmit,
}: {
  query: string;
  placeholder: string;
  onQueryChange: (query: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
}) {
  return (
    <Input
      value={query}
      aria-label={INTAKE_SETTINGS_COPY.labelInput}
      placeholder={placeholder}
      onChange={(event) => onQueryChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          onCancel();
          return;
        }
        if (event.key === "Enter") {
          event.preventDefault();
          onSubmit();
        }
      }}
      data-testid="intake-label-input"
    />
  );
}

function IntakeLabelsAllSelected({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsAllSelected}</p>
      <Button type="button" variant="outline" size="sm" onClick={onAdd}>
        <Plus className="size-3.5" aria-hidden />
        {INTAKE_SETTINGS_COPY.labelAdd}
      </Button>
    </div>
  );
}

function IntakeCreateLabel({ label, onCreate }: { label: string; onCreate: () => void }) {
  return (
    <div>
      <Button type="button" variant="outline" size="sm" onClick={onCreate} data-testid="intake-label-create">
        <Plus className="size-3.5" aria-hidden />
        {`Add label "${label}"`}
      </Button>
    </div>
  );
}

function IntakeLabelOption({ label, onSelect }: { label: string; onSelect: () => void }) {
  return (
    <li>
      <div
        role="checkbox"
        aria-checked={false}
        aria-label={label}
        tabIndex={0}
        className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-muted-foreground hover:bg-accent/50 hover:text-foreground"
        onClick={onSelect}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            onSelect();
          }
        }}
      >
        <Checkbox checked={false} readOnly tabIndex={-1} aria-hidden className="pointer-events-none" />
        <span className="min-w-0 truncate">{label}</span>
      </div>
    </li>
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
