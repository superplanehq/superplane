import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Plus } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { JiraCompletionColumnFields } from "./JiraCompletionColumnFields";
import { addIntakeLabel, toggleIntakeLabel, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const JIRA_INTAKE_SETTINGS_COPY = {
  intakeSection: "Create task when:",
  filtersLabel: "Filters",
  newIssues: "A new issue is created",
  updatedIssues: "An issue is updated",
  filterByLabel: "Issue has one of these labels",
  labelInput: "Issue label",
  labelPlaceholder: "Type a label name",
  labelNew: "Add label",
  labelAdd: "Add",
  labelCancel: "Cancel",
  assignmentAssigned: "Issue is assigned",
  assignmentUnassigned: "Issue is unassigned",
  assignmentAny: "Any assignment",
} as const;

export function JiraIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
  projectId,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
  projectId?: string;
}) {
  if (sourceId !== "jira-issues") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
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
        <legend className="workspace-section-title">{JIRA_INTAKE_SETTINGS_COPY.intakeSection}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <IntakeSettingsCheckbox
            title={JIRA_INTAKE_SETTINGS_COPY.newIssues}
            checked={settings.newIssues}
            onChange={() => update("newIssues", !settings.newIssues)}
          />
          <IntakeSettingsCheckbox
            title={JIRA_INTAKE_SETTINGS_COPY.updatedIssues}
            checked={settings.reopenedIssues}
            onChange={() => update("reopenedIssues", !settings.reopenedIssues)}
          />
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{JIRA_INTAKE_SETTINGS_COPY.filtersLabel}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <div className="flex flex-col gap-1.5">
            <IntakeSettingsCheckbox
              title={JIRA_INTAKE_SETTINGS_COPY.filterByLabel}
              checked={settings.filterByLabel}
              onChange={() => toggleFilterByLabel()}
            />
            {settings.filterByLabel ? (
              <IntakeLabelField labels={settings.labels} onChange={(labels) => update("labels", labels)} />
            ) : null}
          </div>
        </div>
      </fieldset>
      <JiraCompletionColumnFields
        organizationId={organizationId ?? ""}
        integrationId={integrationId ?? ""}
        projectId={projectId ?? ""}
        value={{
          jiraMoveOnComplete: settings.jiraMoveOnComplete,
          jiraCompletionColumn: settings.jiraCompletionColumn,
        }}
        onChange={(next) =>
          onSettingsChange((current) => ({
            ...current,
            jiraMoveOnComplete: next.jiraMoveOnComplete,
            jiraCompletionColumn: next.jiraCompletionColumn,
          }))
        }
      />
    </div>
  );
}

function IntakeLabelField({ labels, onChange }: { labels: string[]; onChange: (labels: string[]) => void }) {
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

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
    <div className="mb-1 ml-6 flex flex-col gap-2" data-testid="jira-intake-label-options">
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
            aria-label={JIRA_INTAKE_SETTINGS_COPY.labelInput}
            placeholder={JIRA_INTAKE_SETTINGS_COPY.labelPlaceholder}
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
            data-testid="jira-intake-label-input"
          />
          <Button
            type="button"
            variant="outline"
            className="shrink-0"
            disabled={draft.trim().length === 0}
            onClick={() => addDraft()}
          >
            {JIRA_INTAKE_SETTINGS_COPY.labelAdd}
          </Button>
          <Button type="button" variant="ghost" className="shrink-0" onClick={() => close()}>
            {JIRA_INTAKE_SETTINGS_COPY.labelCancel}
          </Button>
        </div>
      ) : (
        <div>
          <Button type="button" variant="outline" size="sm" onClick={() => setTyping(true)}>
            <Plus className="size-3.5" aria-hidden />
            {JIRA_INTAKE_SETTINGS_COPY.labelNew}
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
