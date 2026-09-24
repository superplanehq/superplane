import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { cn } from "@/lib/utils";
import { useLayoutEffect, useMemo } from "react";

import { JIRA_COMPLETION_COLUMN_COPY } from "./jiraCompletionColumnCopy";
import { preferredJiraCompletionColumn, type JiraCompletionColumnValue } from "./jiraCompletionColumn";

export function JiraCompletionColumnFields({
  organizationId,
  integrationId,
  projectId,
  value,
  onChange,
}: {
  organizationId: string;
  integrationId: string;
  projectId: string;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
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

  return (
    <fieldset className="min-w-0" data-testid="jira-completion-column">
      <legend className="workspace-section-title">{JIRA_COMPLETION_COLUMN_COPY.section}</legend>
      <div className="mt-2 flex flex-col gap-2">
        <div
          className={cn(
            "flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
            value.jiraMoveOnComplete
              ? "border-foreground/20 bg-accent/50"
              : "border-border bg-card hover:border-foreground/15",
          )}
        >
          <Label htmlFor="jira-move-on-complete" className="flex min-w-0 flex-1 cursor-pointer items-center gap-3">
            <Checkbox
              id="jira-move-on-complete"
              checked={value.jiraMoveOnComplete}
              onChange={() => onChange({ ...value, jiraMoveOnComplete: !value.jiraMoveOnComplete })}
              data-testid="jira-move-on-complete"
            />
            <span className="min-w-0 text-[13px] font-medium tracking-[-0.01em] text-foreground">
              {JIRA_COMPLETION_COLUMN_COPY.move}
            </span>
          </Label>
          <JiraCompletionColumnSelect
            columns={columns}
            selectedColumn={selectedColumn}
            loading={statusesQuery.isLoading}
            disabled={!value.jiraMoveOnComplete}
            onChange={(column) => onChange({ ...value, jiraCompletionColumn: column })}
          />
        </div>
        {value.jiraMoveOnComplete && statusesQuery.isError ? (
          <p className="text-[12px] text-destructive">{JIRA_COMPLETION_COLUMN_COPY.empty}</p>
        ) : null}
      </div>
    </fieldset>
  );
}

function JiraCompletionColumnSelect({
  columns,
  selectedColumn,
  loading,
  disabled = false,
  onChange,
}: {
  columns: string[];
  selectedColumn: string;
  loading: boolean;
  disabled?: boolean;
  onChange: (column: string) => void;
}) {
  const options = columns.map((column) => (
    <SelectItem key={column} value={column}>
      {column}
    </SelectItem>
  ));

  if (selectedColumn) {
    return (
      <Select value={selectedColumn} disabled={disabled} onValueChange={onChange}>
        <SelectTrigger
          id="jira-completion-column-select"
          aria-label={JIRA_COMPLETION_COLUMN_COPY.column}
          className="h-8 w-[min(100%,10rem)] shrink-0"
          data-testid="jira-completion-column-select"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>{options}</SelectContent>
      </Select>
    );
  }

  return (
    <Select disabled={disabled || columns.length === 0} onValueChange={onChange}>
      <SelectTrigger
        id="jira-completion-column-select"
        aria-label={JIRA_COMPLETION_COLUMN_COPY.column}
        className="h-8 w-[min(100%,10rem)] shrink-0"
        data-testid="jira-completion-column-select"
      >
        <SelectValue placeholder={loading ? JIRA_COMPLETION_COLUMN_COPY.loading : JIRA_COMPLETION_COLUMN_COPY.column} />
      </SelectTrigger>
      {columns.length > 0 ? <SelectContent>{options}</SelectContent> : null}
    </Select>
  );
}
