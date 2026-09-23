import { AutoCompleteSelect } from "@/components/AutoCompleteSelect/AutoCompleteSelect";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useLayoutEffect, useMemo } from "react";

import { JIRA_COMPLETION_COLUMN_COPY } from "./jiraCompletionColumnCopy";
import { preferredJiraCompletionColumn, type JiraCompletionColumnValue } from "./jiraCompletionColumn";

export function JiraCompletionColumnFields({
  organizationId,
  integrationId,
  projectId,
  value,
  onChange,
  layout = "boxed",
  showSection = true,
}: {
  organizationId: string;
  integrationId: string;
  projectId: string;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
  /** `plain` is a form section without a nested option card. */
  layout?: "boxed" | "plain";
  /** False when the page heading already names this section. */
  showSection?: boolean;
}) {
  const enabled = Boolean(organizationId && integrationId && projectId);
  const statusesQuery = useIntegrationResources(
    organizationId,
    integrationId,
    "issueStatus",
    enabled ? { project: projectId } : undefined,
    { enabled },
  );
  const columns = useMemo(
    () =>
      (statusesQuery.data ?? [])
        .map((resource) => resource.name?.trim() || resource.id?.trim() || "")
        .filter((name) => name.length > 0),
    [statusesQuery.data],
  );

  const selectedColumn = preferredJiraCompletionColumn(columns, value.jiraCompletionColumn);

  useLayoutEffect(() => {
    if (!value.jiraMoveOnComplete || !selectedColumn) {
      return;
    }
    if (selectedColumn !== value.jiraCompletionColumn) {
      onChange({ jiraMoveOnComplete: value.jiraMoveOnComplete, jiraCompletionColumn: selectedColumn });
    }
  }, [onChange, selectedColumn, value.jiraCompletionColumn, value.jiraMoveOnComplete]);

  if (!projectId) {
    return null;
  }

  const columnFields = value.jiraMoveOnComplete ? (
    <JiraColumnOptions
      columns={columns}
      selectedColumn={selectedColumn}
      loading={statusesQuery.isLoading}
      error={statusesQuery.isError}
      label={layout === "plain" ? JIRA_COMPLETION_COLUMN_COPY.jiraColumn : JIRA_COMPLETION_COLUMN_COPY.column}
      onChange={(column) => onChange({ ...value, jiraCompletionColumn: column })}
    />
  ) : null;

  if (layout === "plain") {
    return (
      <div
        className={cn("overflow-hidden rounded-lg border border-border", showSection ? "mt-2" : "mt-3")}
        data-testid="jira-completion-column"
      >
        {showSection ? (
          <p className="workspace-section-title px-3 pt-3">{JIRA_COMPLETION_COLUMN_COPY.section}</p>
        ) : null}
        <UpdateIssueSwitch checked={value.jiraMoveOnComplete} onChange={onChange} value={value} />
        {columnFields ? <div className="px-3 pb-3 pt-1">{columnFields}</div> : null}
      </div>
    );
  }

  return (
    <fieldset className="min-w-0" data-testid="jira-completion-column">
      {showSection ? <legend className="workspace-section-title">{JIRA_COMPLETION_COLUMN_COPY.section}</legend> : null}
      <div className="mt-2 flex flex-col gap-2">
        <MoveOnCompleteControl checked={value.jiraMoveOnComplete} onChange={onChange} value={value} />
        {columnFields ? <div className="pl-1">{columnFields}</div> : null}
      </div>
    </fieldset>
  );
}

function JiraColumnOptions({
  columns,
  selectedColumn,
  loading,
  error,
  label,
  onChange,
}: {
  columns: string[];
  selectedColumn: string;
  loading: boolean;
  error: boolean;
  label: string;
  onChange: (column: string) => void;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor="jira-completion-column-select">{label}</Label>
      <JiraCompletionColumnSelect
        columns={columns}
        selectedColumn={selectedColumn}
        loading={loading}
        onChange={onChange}
      />
      <p className="text-[12px] text-muted-foreground">{JIRA_COMPLETION_COLUMN_COPY.helper}</p>
      {error ? <p className="text-[12px] text-destructive">{JIRA_COMPLETION_COLUMN_COPY.empty}</p> : null}
    </div>
  );
}

function UpdateIssueSwitch({
  checked,
  value,
  onChange,
}: {
  checked: boolean;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={JIRA_COMPLETION_COLUMN_COPY.update}
      data-testid="jira-move-on-complete"
      data-active={checked ? "true" : "false"}
      onClick={() => onChange({ ...value, jiraMoveOnComplete: !value.jiraMoveOnComplete })}
      className="flex w-full items-center gap-4 px-3 py-2.5 text-left hover:bg-accent/40"
    >
      <span className="min-w-0 flex-1">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {JIRA_COMPLETION_COLUMN_COPY.update}
        </span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">
          {JIRA_COMPLETION_COLUMN_COPY.updateDescription}
        </span>
      </span>
      <span
        aria-hidden
        className={cn(
          "relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors",
          checked ? "bg-foreground" : "bg-muted-foreground/30",
        )}
      >
        <span
          className={cn(
            "absolute top-0.5 size-4 rounded-full bg-background shadow-sm transition-transform",
            checked ? "translate-x-4" : "translate-x-0.5",
          )}
        />
      </span>
    </button>
  );
}

function MoveOnCompleteControl({
  checked,
  value,
  onChange,
}: {
  checked: boolean;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
}) {
  const toggle = () => onChange({ ...value, jiraMoveOnComplete: !value.jiraMoveOnComplete });

  return (
    <Label
      htmlFor="jira-move-on-complete"
      className={cn(
        "flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox id="jira-move-on-complete" checked={checked} onChange={toggle} data-testid="jira-move-on-complete" />
      <span className="min-w-0 text-[13px] font-medium tracking-[-0.01em] text-foreground">
        {JIRA_COMPLETION_COLUMN_COPY.move}
      </span>
    </Label>
  );
}

function JiraCompletionColumnSelect({
  columns,
  selectedColumn,
  loading,
  onChange,
}: {
  columns: string[];
  selectedColumn: string;
  loading: boolean;
  onChange: (column: string) => void;
}) {
  const options = columns.map((column) => ({ value: column, label: column }));

  return (
    <AutoCompleteSelect
      id="jira-completion-column-select"
      testId="jira-completion-column-select"
      options={options}
      value={selectedColumn}
      onChange={onChange}
      placeholder={loading ? JIRA_COMPLETION_COLUMN_COPY.loading : JIRA_COMPLETION_COLUMN_COPY.column}
      disabled={columns.length === 0 && !loading}
    />
  );
}
