import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Plus, X } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { IntakeEventRow } from "./IntakeEventRow";
import { JiraCompletionColumnFields } from "./JiraCompletionColumnFields";
import { JIRA_COMPLETION_COLUMN_COPY } from "./jiraCompletionColumnCopy";
import {
  addIntakeLabel,
  INTAKE_SETTINGS_COPY,
  intakeSettingsSectionDomId,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const JIRA_INTAKE_EVENTS = [
  {
    key: "newIssues",
    title: "A new issue is created",
    description: "Jira adds an issue.",
    testId: "jira-intake-new-issues",
  },
  {
    key: "reopenedIssues",
    title: "An issue is updated",
    description: "An existing issue changes.",
    testId: "jira-intake-updated-issues",
  },
] as const satisfies ReadonlyArray<{
  key: "newIssues" | "reopenedIssues";
  title: string;
  description: string;
  testId: string;
}>;

const JIRA_INTAKE_SETTINGS_COPY = {
  labelsSection: "Only intake issues with these labels",
  labelsHelper: "Leave empty to intake every issue.",
  labelInput: "Issue label",
  labelPlaceholder: "Type a label name",
  labelNew: "Add label",
  labelAdd: "Add",
  labelCancel: "Cancel",
  factorySection: JIRA_COMPLETION_COLUMN_COPY.section,
  factoryHelper: "Choose whether SuperPlane updates the Jira issue.",
} as const;

export function JiraIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
  projectId,
  part = "all",
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
  projectId?: string;
  /** Which block to show. The settings sidebar shows one block at a time. */
  part?: "all" | "create" | "filters" | "complete";
  /** Kept for callers that still pass a layout. The event list is always a stack. */
  layout?: "stack" | "grid";
}) {
  if (sourceId !== "jira-issues") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  const showIntake = part === "all" || part === "create" || part === "filters";
  const showFactory = part === "all" || part === "complete";

  return (
    <div className="flex flex-col gap-8">
      {showIntake ? (
        <section
          id={intakeSettingsSectionDomId("triggers")}
          className="scroll-mt-6 min-w-0"
          data-testid="jira-intake-triggers"
        >
          <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.eventsThatCreateTasks}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{INTAKE_SETTINGS_COPY.eventsThatCreateTasksHelper}</p>
          <div
            className="mt-3 divide-y divide-border overflow-hidden rounded-lg border border-border"
            role="group"
            aria-label={INTAKE_SETTINGS_COPY.eventsThatCreateTasks}
          >
            {JIRA_INTAKE_EVENTS.map((event) => (
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
      {showIntake ? (
        <section
          id={intakeSettingsSectionDomId("labels")}
          className="scroll-mt-6 min-w-0"
          data-testid="jira-intake-labels"
        >
          <h3 className="workspace-section-title">{JIRA_INTAKE_SETTINGS_COPY.labelsSection}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{JIRA_INTAKE_SETTINGS_COPY.labelsHelper}</p>
          <IntakeLabelField
            labels={settings.labels}
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
      {showFactory ? (
        <section
          id={intakeSettingsSectionDomId("factory")}
          className="scroll-mt-6 min-w-0"
          data-testid="jira-intake-factory"
        >
          <h3 className="workspace-section-title">{JIRA_INTAKE_SETTINGS_COPY.factorySection}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">{JIRA_INTAKE_SETTINGS_COPY.factoryHelper}</p>
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
            layout="plain"
            showSection={false}
          />
        </section>
      ) : null}
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

  function remove(label: string) {
    onChange(labels.filter((entry) => entry !== label));
  }

  return (
    <div className="mt-2 flex flex-col gap-2" data-testid="jira-intake-label-options">
      {labels.length > 0 ? (
        <ul className="flex flex-wrap items-center gap-1.5">
          {labels.map((label) => (
            <li key={label}>
              <span
                className={cn(
                  "inline-flex max-w-full items-center gap-1 rounded-md border px-2 py-1 text-[13px]",
                  "border-foreground/20 bg-accent/50 text-foreground",
                )}
              >
                <span className="min-w-0 truncate">{label}</span>
                <button
                  type="button"
                  className="rounded-sm text-muted-foreground hover:text-foreground"
                  aria-label={`Remove ${label}`}
                  onClick={() => remove(label)}
                >
                  <X className="size-3.5" aria-hidden />
                </button>
              </span>
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
            data-testid="jira-intake-label-add"
          >
            {JIRA_INTAKE_SETTINGS_COPY.labelAdd}
          </Button>
          <Button
            type="button"
            variant="ghost"
            className="shrink-0"
            onClick={() => close()}
            data-testid="jira-intake-label-cancel"
          >
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
