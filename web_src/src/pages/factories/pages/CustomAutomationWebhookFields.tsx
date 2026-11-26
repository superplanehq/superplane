import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFactoryWebhooks } from "@/hooks/useFactoryWebhooks";
import { CopyButton } from "@/ui/CopyButton";
import { HighlightedCode } from "@/ui/HighlightedCode";

import type { CustomAutomationTaskMatch } from "../lib/customAutomation";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { validWebhookIntakeFieldPath } from "./webhookIntakeFieldPaths";
import { WEBHOOK_INTAKE_FIELDS_COPY } from "./webhookIntakeSetupCopy";

export function CustomAutomationWebhookFields({
  idPrefix,
  organizationId,
  factoryId,
  factoryKey,
  webhookId,
  usedWebhookIds,
  onChange,
  helper = "The agent gets the JSON request body and can create a task.",
}: {
  idPrefix: string;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  webhookId?: string;
  /** Webhooks that other triggers of this automation already use. */
  usedWebhookIds: string[];
  onChange: (webhookId: string) => void;
  helper?: string;
}) {
  const webhooks = useFactoryWebhooks(organizationId, factoryId);
  const settingsPath = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "webhooks");
  const options = (webhooks.data ?? []).filter((webhook) => webhook.id && !usedWebhookIds.includes(webhook.id));
  const selected = webhooks.data?.find((webhook) => webhook.id === webhookId);

  if (webhooks.isLoading) {
    return <p className="text-xs text-muted-foreground">Loading webhooks...</p>;
  }
  if (webhooks.isError) {
    return (
      <p className="text-xs text-muted-foreground" role="alert">
        SuperPlane could not load the webhooks.{" "}
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto p-0 text-xs"
          onClick={() => void webhooks.refetch()}
        >
          Try again
        </Button>
      </p>
    );
  }
  if (options.length === 0 && !selected) {
    return (
      <p className="text-xs text-muted-foreground" data-testid="custom-automation-webhook-empty">
        This workspace has no available webhooks.{" "}
        <Link to={settingsPath} className="font-medium text-foreground underline underline-offset-2">
          Create a webhook in settings
        </Link>
      </p>
    );
  }
  return (
    <div className="space-y-3">
      <Label htmlFor={`${idPrefix}-webhook`}>Webhook</Label>
      <Select value={webhookId ?? ""} onValueChange={onChange}>
        <SelectTrigger id={`${idPrefix}-webhook`} className="w-full" data-testid="custom-automation-webhook-select">
          <SelectValue placeholder="Select a webhook" />
        </SelectTrigger>
        <SelectContent>
          {[...(selected ? [selected] : []), ...options.filter((webhook) => webhook.id !== selected?.id)].map(
            (webhook) => (
              <SelectItem key={webhook.id} value={webhook.id!}>
                {webhook.name}
              </SelectItem>
            ),
          )}
        </SelectContent>
      </Select>
      {selected?.url ? (
        <div className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2">
          <code className="min-w-0 flex-1 truncate text-left text-[12px] text-foreground">{selected.url}</code>
          <CopyButton text={selected.url} ariaLabel="Copy webhook URL" copiedAriaLabel="Webhook URL copied" />
        </div>
      ) : null}
      <p className="text-xs text-muted-foreground">
        {helper}{" "}
        <Link
          to={settingsPath}
          target="_blank"
          rel="noreferrer"
          className="font-medium text-foreground underline underline-offset-2"
        >
          Manage webhooks
        </Link>
      </p>
    </div>
  );
}

export function TaskMatchFields({
  idPrefix,
  taskMatch,
  onChange,
}: {
  idPrefix: string;
  taskMatch: CustomAutomationTaskMatch;
  onChange: (taskMatch: CustomAutomationTaskMatch) => void;
}) {
  const example = taskMatchExampleBody(taskMatch);
  return (
    <div className="mt-4 grid gap-4 border-t border-border pt-4">
      <div className="grid gap-2">
        <Label htmlFor={`${idPrefix}-task-match-kind`}>Find task by</Label>
        <Select
          value={taskMatch.kind}
          onValueChange={(kind: CustomAutomationTaskMatch["kind"]) => onChange({ ...taskMatch, kind })}
        >
          <SelectTrigger id={`${idPrefix}-task-match-kind`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="task">Task</SelectItem>
            <SelectItem value="pull_request">Pull request</SelectItem>
            <SelectItem value="commit">Commit</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="grid gap-2">
        <Label htmlFor={`${idPrefix}-task-match-path`}>JSON body field</Label>
        <Input
          id={`${idPrefix}-task-match-path`}
          value={taskMatch.fieldPath}
          aria-describedby={`${idPrefix}-task-match-path-help`}
          placeholder={
            taskMatch.kind === "task"
              ? "issue.key"
              : taskMatch.kind === "pull_request"
                ? "pull_request.url"
                : "commit.sha"
          }
          onChange={(event) => onChange({ ...taskMatch, fieldPath: event.target.value })}
        />
        <p id={`${idPrefix}-task-match-path-help`} className="text-xs text-muted-foreground">
          {taskMatchPathHelper(taskMatch.kind)}
        </p>
        {example ? (
          <div className="grid gap-2">
            <p className="text-xs text-muted-foreground">{WEBHOOK_INTAKE_FIELDS_COPY.exampleLabel}</p>
            <HighlightedCode
              code={example}
              language="json"
              className="rounded-md border border-border bg-muted/40 px-3 py-2"
              testId={`${idPrefix}-task-match-example`}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}

function taskMatchPathHelper(kind: CustomAutomationTaskMatch["kind"]): string {
  if (kind === "pull_request") {
    return "The value is a pull request URL. Use dots for nested fields, for example pull_request.url.";
  }
  if (kind === "commit") {
    return "The value is a commit SHA. Use dots for nested fields, for example commit.sha.";
  }
  return "The value is a task id, number, or key. Use dots for nested fields, for example issue.key.";
}

function taskMatchExampleBody(taskMatch: CustomAutomationTaskMatch): string {
  const path = taskMatch.fieldPath.trim();
  if (!validWebhookIntakeFieldPath(path)) return "";
  const value =
    taskMatch.kind === "pull_request"
      ? "https://github.com/acme/app/pull/40"
      : taskMatch.kind === "commit"
        ? "abc123def456"
        : "SP-42";
  const body: Record<string, unknown> = Object.create(null);
  const segments = path.split(".");
  let parent = body;
  for (const segment of segments.slice(0, -1)) {
    const child: Record<string, unknown> = Object.create(null);
    parent[segment] = child;
    parent = child;
  }
  parent[segments[segments.length - 1]] = value;
  return JSON.stringify(body, null, 2);
}
