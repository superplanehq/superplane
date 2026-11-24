import { Link } from "react-router";

import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFactoryWebhooks } from "@/hooks/useFactoryWebhooks";
import { CopyButton } from "@/ui/CopyButton";

import { factorySettingsSectionPath } from "../lib/factoryPagePaths";

export function CustomAutomationWebhookFields({
  idPrefix,
  organizationId,
  factoryId,
  factoryKey,
  webhookId,
  usedWebhookIds,
  onChange,
}: {
  idPrefix: string;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  webhookId?: string;
  /** Webhooks that other triggers of this automation already use. */
  usedWebhookIds: string[];
  onChange: (webhookId: string) => void;
}) {
  const webhooks = useFactoryWebhooks(organizationId, factoryId);
  const settingsPath = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "webhooks");
  const options = (webhooks.data ?? []).filter((webhook) => webhook.id && !usedWebhookIds.includes(webhook.id));
  const selected = webhooks.data?.find((webhook) => webhook.id === webhookId);

  if (webhooks.isLoading) {
    return <p className="text-xs text-muted-foreground">Loading webhooks...</p>;
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
        The agent gets the JSON request body and can create a task.{" "}
        <Link to={settingsPath} className="font-medium text-foreground underline underline-offset-2">
          Manage webhooks
        </Link>
      </p>
    </div>
  );
}
