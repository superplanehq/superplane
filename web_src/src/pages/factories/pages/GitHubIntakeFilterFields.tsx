import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Plus, X } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type RefObject, type SetStateAction } from "react";

import { IntakeEventRow } from "./IntakeEventRow";
import {
  addIntakeLabel,
  INTAKE_SETTINGS_COPY,
  intakeSettingsSectionDomId,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const GITHUB_INTAKE_EVENTS = [
  {
    key: "newIssues",
    title: INTAKE_SETTINGS_COPY.newIssues,
    description: "GitHub opens an issue.",
    testId: "github-intake-new-issues",
  },
  {
    key: "reopenedIssues",
    title: INTAKE_SETTINGS_COPY.reopenedIssues,
    description: "GitHub opens a closed issue again.",
    testId: "github-intake-reopened-issues",
  },
  {
    key: "superplaneLabelAdded",
    title: INTAKE_SETTINGS_COPY.superplaneLabelAdded,
    description: "A person adds that label to an open issue.",
    testId: "github-intake-superplane-label",
  },
] as const satisfies ReadonlyArray<{
  key: "newIssues" | "reopenedIssues" | "superplaneLabelAdded";
  title: string;
  description: string;
  testId: string;
}>;

const GITHUB_INTAKE_SETTINGS_COPY = {
  labelsSection: "Only intake issues with these labels",
  labelsHelper: "Leave empty to intake every issue.",
  labelSuggestions: "Labels in the repository",
  filtersHelper: "These rules limit which issues create tasks.",
  authorsWithAccessDescription: "The issue author has access to the repository.",
} as const;

/** Repository labels that match the text the user has typed. */
function githubLabelSuggestions(options: string[], selected: string[], query: string): string[] {
  const needle = query.trim().toLowerCase();
  if (needle.length === 0) {
    return [];
  }
  return options.filter((label) => !selected.includes(label) && label.toLowerCase().includes(needle));
}

export function GitHubIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  labelOptions = [],
  labelOptionsLoading = false,
  part = "all",
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  /** Labels that exist in the connected repository. */
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  /** Which block to show. The settings sidebar shows one block at a time. */
  part?: "all" | "create" | "filters";
  /** Kept for callers that still pass a layout. The event list is always a stack. */
  layout?: "stack" | "grid";
}) {
  if (sourceId !== "github-issues") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  const showEvents = part === "all" || part === "create";
  const showFilters = part === "all" || part === "filters";

  return (
    <div className="flex flex-col gap-8">
      {showEvents ? (
        <section
          id={intakeSettingsSectionDomId("triggers")}
          className="scroll-mt-6 min-w-0"
          data-testid="github-intake-triggers"
        >
          <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.eventsThatCreateTasks}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{INTAKE_SETTINGS_COPY.eventsThatCreateTasksHelper}</p>
          <div
            className="mt-3 divide-y divide-border overflow-hidden rounded-lg border border-border"
            role="group"
            aria-label={INTAKE_SETTINGS_COPY.eventsThatCreateTasks}
          >
            {GITHUB_INTAKE_EVENTS.map((event) => (
              <IntakeEventRow
                key={event.key}
                title={event.title}
                description={event.description}
                active={settings[event.key]}
                onToggle={() => update(event.key, !settings[event.key])}
                testId={event.testId}
              />
            ))}
          </div>
        </section>
      ) : null}
      {showFilters ? (
        <section
          id={intakeSettingsSectionDomId("labels")}
          className="scroll-mt-6 min-w-0"
          data-testid="github-intake-labels"
        >
          <h3 className="workspace-section-title">{GITHUB_INTAKE_SETTINGS_COPY.labelsSection}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{GITHUB_INTAKE_SETTINGS_COPY.labelsHelper}</p>
          <IntakeLabelField
            labels={settings.labels}
            options={labelOptions}
            loading={labelOptionsLoading}
            onChange={(labels) =>
              onSettingsChange((current) => ({
                ...current,
                labels,
                filterByLabel: labels.length > 0,
                labelFilterMode: labels.length > 0 ? current.labelFilterMode : "include",
              }))
            }
          />
        </section>
      ) : null}
      {showFilters ? (
        <section
          id={intakeSettingsSectionDomId("filters")}
          className="scroll-mt-6 min-w-0"
          data-testid="github-intake-filters"
        >
          <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.filtersLabel}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{GITHUB_INTAKE_SETTINGS_COPY.filtersHelper}</p>
          <div
            className="mt-3 divide-y divide-border overflow-hidden rounded-lg border border-border"
            role="group"
            aria-label={INTAKE_SETTINGS_COPY.filtersLabel}
          >
            <IntakeEventRow
              title={INTAKE_SETTINGS_COPY.authorsWithAccess}
              description={GITHUB_INTAKE_SETTINGS_COPY.authorsWithAccessDescription}
              active={settings.authorsWithAccess}
              onToggle={() => update("authorsWithAccess", !settings.authorsWithAccess)}
              testId="github-intake-authors-with-access"
            />
          </div>
        </section>
      ) : null}
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
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestions = githubLabelSuggestions(options, labels, draft);

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
    <div className="mt-2 flex flex-col gap-2" data-testid="intake-label-options">
      <GitHubSelectedLabels labels={labels} onRemove={(label) => onChange(labels.filter((entry) => entry !== label))} />
      {typing ? (
        <GitHubLabelDraft
          draft={draft}
          loading={loading}
          suggestions={suggestions}
          inputRef={inputRef}
          onDraftChange={setDraft}
          onAddDraft={() => addDraft()}
          onClose={() => close()}
          onPick={(label) => {
            add(label);
            close();
          }}
        />
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

function GitHubSelectedLabels({ labels, onRemove }: { labels: string[]; onRemove: (label: string) => void }) {
  if (labels.length === 0) {
    return null;
  }

  return (
    <ul className="flex flex-wrap items-center gap-1.5">
      {labels.map((label) => (
        <li key={label}>
          <span className="inline-flex max-w-full items-center gap-1 rounded-md border border-foreground/20 bg-accent/50 px-2 py-1 text-[13px] text-foreground">
            <span className="min-w-0 truncate">{label}</span>
            <button
              type="button"
              className="rounded-sm text-muted-foreground hover:text-foreground"
              aria-label={`Remove ${label}`}
              onClick={() => onRemove(label)}
            >
              <X className="size-3.5" aria-hidden />
            </button>
          </span>
        </li>
      ))}
    </ul>
  );
}

function GitHubLabelDraft({
  draft,
  loading,
  suggestions,
  inputRef,
  onDraftChange,
  onAddDraft,
  onClose,
  onPick,
}: {
  draft: string;
  loading: boolean;
  suggestions: string[];
  inputRef: RefObject<HTMLInputElement | null>;
  onDraftChange: (value: string) => void;
  onAddDraft: () => void;
  onClose: () => void;
  onPick: (label: string) => void;
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <Input
          ref={inputRef}
          value={draft}
          aria-label={INTAKE_SETTINGS_COPY.labelInput}
          aria-autocomplete="list"
          aria-expanded={suggestions.length > 0}
          placeholder={INTAKE_SETTINGS_COPY.labelPlaceholder}
          onChange={(event) => onDraftChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              onClose();
              return;
            }
            if (event.key !== "Enter") {
              return;
            }
            event.preventDefault();
            onAddDraft();
          }}
          data-testid="intake-label-input"
        />
        <Button
          type="button"
          variant="outline"
          className="shrink-0"
          disabled={draft.trim().length === 0}
          onClick={() => onAddDraft()}
          data-testid="intake-label-add"
        >
          {INTAKE_SETTINGS_COPY.labelAdd}
        </Button>
        <Button
          type="button"
          variant="ghost"
          className="shrink-0"
          onClick={() => onClose()}
          data-testid="intake-label-cancel"
        >
          {INTAKE_SETTINGS_COPY.labelCancel}
        </Button>
      </div>
      {loading && draft.trim().length > 0 ? (
        <p className="text-[12px] text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsLoading}</p>
      ) : null}
      <GitHubLabelSuggestions suggestions={suggestions} onPick={(label) => onPick(label)} />
    </div>
  );
}

function GitHubLabelSuggestions({ suggestions, onPick }: { suggestions: string[]; onPick: (label: string) => void }) {
  if (suggestions.length === 0) {
    return null;
  }

  return (
    <ul
      role="listbox"
      aria-label={GITHUB_INTAKE_SETTINGS_COPY.labelSuggestions}
      className="max-h-40 overflow-y-auto rounded-lg border border-border bg-card py-1"
      data-testid="intake-label-suggestions"
    >
      {suggestions.map((label) => (
        <li key={label}>
          <button
            type="button"
            role="option"
            aria-selected={false}
            className="flex w-full px-3 py-1.5 text-left text-[13px] text-foreground hover:bg-accent/60"
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => onPick(label)}
          >
            {label}
          </button>
        </li>
      ))}
    </ul>
  );
}
