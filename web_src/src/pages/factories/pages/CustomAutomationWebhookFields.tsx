import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFactoryWebhooks } from "@/hooks/useFactoryWebhooks";
import { CopyButton } from "@/ui/CopyButton";

import type { CustomAutomationTaskMatch } from "../lib/customAutomation";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";

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
        <Link to={settingsPath} className="font-medium text-foreground underline underline-offset-2">
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
        <Label htmlFor={`${idPrefix}-task-match-path`}>Field path</Label>
        <Input
          id={`${idPrefix}-task-match-path`}
          value={taskMatch.fieldPath}
          placeholder={
            taskMatch.kind === "task"
              ? "task.key"
              : taskMatch.kind === "pull_request"
                ? "pull_request.url"
                : "commit.sha"
          }
          onChange={(event) => onChange({ ...taskMatch, fieldPath: event.target.value })}
        />
      </div>
    </div>
  );
}
