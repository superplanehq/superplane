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
  layout = "boxed",
}: {
  organizationId: string;
  integrationId: string;
  projectId: string;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
  /** `plain` is a form section without a nested option card. */
  layout?: "boxed" | "plain";
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
      <div className={cn("flex flex-col", layout === "plain" ? "mt-3 gap-3" : "mt-2 gap-2")}>
        <MoveOnCompleteControl layout={layout} checked={value.jiraMoveOnComplete} onChange={onChange} value={value} />
        {value.jiraMoveOnComplete ? (
          <div className={cn("flex flex-col gap-1.5", layout === "plain" ? "" : "pl-1")}>
            <Label htmlFor="jira-completion-column-select">{JIRA_COMPLETION_COLUMN_COPY.column}</Label>
            <JiraCompletionColumnSelect
              columns={columns}
              selectedColumn={selectedColumn}
              loading={statusesQuery.isLoading}
              onChange={(column) => onChange({ ...value, jiraCompletionColumn: column })}
            />
            <p className="text-[12px] text-muted-foreground">{JIRA_COMPLETION_COLUMN_COPY.helper}</p>
            {statusesQuery.isError ? (
              <p className="text-[12px] text-destructive">{JIRA_COMPLETION_COLUMN_COPY.empty}</p>
            ) : null}
          </div>
        ) : null}
      </div>
    </fieldset>
  );
}

function MoveOnCompleteControl({
  layout,
  checked,
  value,
  onChange,
}: {
  layout: "boxed" | "plain";
  checked: boolean;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
}) {
  const toggle = () => onChange({ ...value, jiraMoveOnComplete: !value.jiraMoveOnComplete });

  if (layout === "plain") {
    return (
      <div className="flex items-start gap-3">
        <Checkbox
          id="jira-move-on-complete"
          className="mt-0.5"
          checked={checked}
          onChange={toggle}
          data-testid="jira-move-on-complete"
        />
        <Label htmlFor="jira-move-on-complete" className="cursor-pointer text-[13px] font-medium tracking-[-0.01em]">
          {JIRA_COMPLETION_COLUMN_COPY.move}
        </Label>
      </div>
    );
  }

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
  const options = columns.map((column) => (
    <SelectItem key={column} value={column}>
      {column}
    </SelectItem>
  ));

  if (selectedColumn) {
    return (
      <Select value={selectedColumn} onValueChange={onChange}>
        <SelectTrigger
          id="jira-completion-column-select"
          className="w-full"
          data-testid="jira-completion-column-select"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>{options}</SelectContent>
      </Select>
    );
  }

  return (
    <Select disabled={columns.length === 0} onValueChange={onChange}>
      <SelectTrigger id="jira-completion-column-select" className="w-full" data-testid="jira-completion-column-select">
        <SelectValue placeholder={loading ? JIRA_COMPLETION_COLUMN_COPY.loading : JIRA_COMPLETION_COLUMN_COPY.column} />
      </SelectTrigger>
      {columns.length > 0 ? <SelectContent>{options}</SelectContent> : null}
    </Select>
  );
}
