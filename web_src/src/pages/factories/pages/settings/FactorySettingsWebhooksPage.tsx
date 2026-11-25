import { useState } from "react";

import type { FactoriesFactoryWebhook, FactoryWebhookAuthentication } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryWebhookMutations, useFactoryWebhooks } from "@/hooks/useFactoryWebhooks";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  DEFAULT_WEBHOOK_HEADER_TOKEN_NAME,
  DEFAULT_WEBHOOK_SIGNATURE_HEADER,
  generateWebhookCode,
} from "@/lib/webhookRequestExample";
import { CopyButton } from "@/ui/CopyButton";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export type Authentication = Exclude<FactoryWebhookAuthentication, "AUTHENTICATION_UNSPECIFIED">;

const AUTHENTICATION_OPTIONS: { value: Authentication; label: string; mode: string }[] = [
  { value: "AUTHENTICATION_BEARER", label: "Bearer token", mode: "bearer" },
  { value: "AUTHENTICATION_SIGNATURE", label: "HMAC signature", mode: "signature" },
  { value: "AUTHENTICATION_HEADER_TOKEN", label: "Header token", mode: "header_token" },
];

const DEFAULT_HEADER: Partial<Record<Authentication, string>> = {
  AUTHENTICATION_SIGNATURE: DEFAULT_WEBHOOK_SIGNATURE_HEADER,
  AUTHENTICATION_HEADER_TOKEN: DEFAULT_WEBHOOK_HEADER_TOKEN_NAME,
};

const NO_UPDATE_PERMISSION = "You need permission to update this workspace.";

type Editor = { webhook?: FactoriesFactoryWebhook };
export type Reveal = { webhook: FactoriesFactoryWebhook; secret: string };

export function FactorySettingsWebhooksPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const webhooks = useFactoryWebhooks(organizationId, factoryId);
  const mutations = useFactoryWebhookMutations(organizationId, factoryId);
  const [editor, setEditor] = useState<Editor | null>(null);
  const [reveal, setReveal] = useState<Reveal | null>(null);
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryWebhook | null>(null);
  const [pendingRegenerate, setPendingRegenerate] = useState<FactoriesFactoryWebhook | null>(null);
  usePageTitle(["Webhooks", "Settings", factory.name ?? "Workspace"]);

  const list = webhooks.data ?? [];

  return (
    <FactorySettingsPageFrame
      title="Webhooks"
      subtitle="Receive requests from other tools. Select a webhook as the trigger of a Backlog automation."
      actions={
        <PermissionTooltip allowed={canUpdate} message={NO_UPDATE_PERMISSION}>
          <Button
            type="button"
            size="sm"
            disabled={!canUpdate}
            onClick={() => setEditor({})}
            data-testid="factory-webhook-create"
          >
            Create webhook
          </Button>
        </PermissionTooltip>
      }
    >
      <FactorySettingsCard attachedList={list.length > 0} data-testid="factory-webhooks">
        {webhooks.isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading webhooks...</p>
        ) : webhooks.isError ? (
          <p className="text-[13px] text-destructive">SuperPlane could not load the webhooks. Refresh the page.</p>
        ) : list.length === 0 ? (
          <Empty className="flex-none gap-3 border-none p-6 md:p-8" data-testid="factory-webhooks-empty">
            <EmptyHeader className="max-w-md gap-3">
              <EmptyTitle className="text-[15px] font-medium">No webhooks</EmptyTitle>
              <EmptyDescription className="text-[13px]">
                Create a webhook to start automations from Jira, monitoring, or deploy tools.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ul className="divide-y divide-border">
            {list.map((webhook) => (
              <WebhookRow
                key={webhook.id}
                webhook={webhook}
                canUpdate={canUpdate}
                onEdit={() => setEditor({ webhook })}
                onRegenerate={() => setPendingRegenerate(webhook)}
                onDelete={() => setPendingDelete(webhook)}
              />
            ))}
          </ul>
        )}
      </FactorySettingsCard>
      <WebhookEditorDialog
        editor={editor}
        mutations={mutations}
        onClose={() => setEditor(null)}
        onCreated={(created) => setReveal(created)}
      />
      <WebhookSecretDialog reveal={reveal} onClose={() => setReveal(null)} />
      <FactoryDeleteDialog
        open={Boolean(pendingRegenerate)}
        factoryName={pendingRegenerate?.name ?? ""}
        title={`Regenerate the secret for "${pendingRegenerate?.name ?? ""}"?`}
        description="The current secret stops working now. Update the sender with the new secret."
        confirmLabel="Regenerate secret"
        loadingText="Regenerating..."
        canDelete={canUpdate}
        isDeleting={mutations.regenerateSecret.isPending}
        onClose={() => setPendingRegenerate(null)}
        onConfirm={async () => {
          if (!pendingRegenerate?.id) {
            return;
          }
          try {
            const secret = await mutations.regenerateSecret.mutateAsync(pendingRegenerate.id);
            setReveal({ webhook: pendingRegenerate, secret });
          } catch (error) {
            showErrorToast(getApiErrorMessage(error) || "SuperPlane could not regenerate the secret.");
            throw error;
          }
        }}
      />
      <FactoryDeleteDialog
        open={Boolean(pendingDelete)}
        factoryName={pendingDelete?.name ?? ""}
        title={`Delete "${pendingDelete?.name ?? ""}"?`}
        description="Requests to this URL stop working. You cannot undo this."
        confirmLabel="Delete webhook"
        canDelete={canUpdate}
        isDeleting={mutations.remove.isPending}
        onClose={() => setPendingDelete(null)}
        onConfirm={async () => {
          if (!pendingDelete?.id) {
            return;
          }
          try {
            await mutations.remove.mutateAsync(pendingDelete.id);
            showSuccessToast("Webhook deleted.");
          } catch (error) {
            showErrorToast(getApiErrorMessage(error) || "SuperPlane could not delete the webhook.");
            throw error;
          }
        }}
      />
    </FactorySettingsPageFrame>
  );
}

function WebhookRow({
  webhook,
  canUpdate,
  onEdit,
  onRegenerate,
  onDelete,
}: {
  webhook: FactoriesFactoryWebhook;
  canUpdate: boolean;
  onEdit: () => void;
  onRegenerate: () => void;
  onDelete: () => void;
}) {
  const usage = webhook.automationCount ?? 0;
  const inUse = usage > 0;
  return (
    <li className="flex flex-col gap-2 py-3" data-testid={`factory-webhook-${webhook.id}`}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-[13px] font-medium text-foreground">{webhook.name}</p>
          <p className="text-[12px] text-muted-foreground">
            {authenticationSummary(webhook)} · {usageLabel(usage)}
          </p>
        </div>
        <div className="flex items-center gap-1">
          <PermissionTooltip allowed={canUpdate} message={NO_UPDATE_PERMISSION}>
            <Button type="button" size="sm" variant="ghost" disabled={!canUpdate} onClick={onEdit}>
              Edit
            </Button>
          </PermissionTooltip>
          <PermissionTooltip allowed={canUpdate} message={NO_UPDATE_PERMISSION}>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={!canUpdate}
              onClick={onRegenerate}
              data-testid={`factory-webhook-regenerate-${webhook.id}`}
            >
              Regenerate secret
            </Button>
          </PermissionTooltip>
          <PermissionTooltip
            allowed={canUpdate && !inUse}
            message={inUse ? "Remove this webhook from its automations before you delete it." : NO_UPDATE_PERMISSION}
          >
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={!canUpdate || inUse}
              onClick={onDelete}
              data-testid={`factory-webhook-delete-${webhook.id}`}
            >
              Delete
            </Button>
          </PermissionTooltip>
        </div>
      </div>
      <CopyField label="Webhook URL" value={webhook.url ?? ""} testId={`factory-webhook-url-${webhook.id}`} />
    </li>
  );
}

function WebhookEditorDialog({
  editor,
  mutations,
  onClose,
  onCreated,
}: {
  editor: Editor | null;
  mutations: ReturnType<typeof useFactoryWebhookMutations>;
  onClose: () => void;
  onCreated: (reveal: Reveal) => void;
}) {
  return (
    <Dialog open={Boolean(editor)} onOpenChange={(open) => !open && onClose()}>
      {editor ? (
        <WebhookEditorForm
          key={editor.webhook?.id ?? "new"}
          webhook={editor.webhook}
          mutations={mutations}
          onClose={onClose}
          onCreated={onCreated}
        />
      ) : null}
    </Dialog>
  );
}

function WebhookEditorForm({
  webhook,
  mutations,
  onClose,
  onCreated,
}: {
  webhook?: FactoriesFactoryWebhook;
  mutations: ReturnType<typeof useFactoryWebhookMutations>;
  onClose: () => void;
  onCreated: (reveal: Reveal) => void;
}) {
  const [name, setName] = useState(webhook?.name ?? "");
  const [authentication, setAuthentication] = useState<Authentication>(
    isAuthentication(webhook?.authentication) ? webhook.authentication : "AUTHENTICATION_BEARER",
  );
  const [headerName, setHeaderName] = useState(webhook?.headerName ?? "");
  const [error, setError] = useState("");
  const headerDefault = DEFAULT_HEADER[authentication];
  const pending = mutations.create.isPending || mutations.update.isPending;

  const submit = async () => {
    if (!name.trim()) {
      setError("Enter a webhook name.");
      return;
    }
    const body = { name: name.trim(), authentication, headerName: headerDefault ? headerName.trim() : "" };
    try {
      if (webhook?.id) {
        await mutations.update.mutateAsync({ webhookId: webhook.id, body });
        showSuccessToast("Webhook updated.");
        onClose();
        return;
      }
      const created = await mutations.create.mutateAsync(body);
      onClose();
      if (created?.webhook && created.secret) {
        onCreated({ webhook: created.webhook, secret: created.secret });
      }
    } catch (submitError) {
      setError(getApiErrorMessage(submitError) || "SuperPlane could not save the webhook.");
    }
  };

  return (
    <DialogContent className="max-w-lg" data-testid="factory-webhook-form">
      <DialogHeader>
        <DialogTitle>{webhook ? "Edit webhook" : "Create webhook"}</DialogTitle>
        <DialogDescription>
          {webhook
            ? "The URL and the secret do not change."
            : "SuperPlane shows the secret once, after you create the webhook."}
        </DialogDescription>
      </DialogHeader>
      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <div className="space-y-2">
          <Label htmlFor="factory-webhook-name">Name</Label>
          <Input
            id="factory-webhook-name"
            value={name}
            placeholder="Jira issues"
            maxLength={100}
            onChange={(event) => {
              setError("");
              setName(event.target.value);
            }}
            data-testid="factory-webhook-name"
          />
        </div>
        <WebhookAuthenticationField
          value={authentication}
          onChange={(value) => {
            setError("");
            setAuthentication(value);
          }}
        />
        {headerDefault ? (
          <div className="space-y-2">
            <Label htmlFor="factory-webhook-header">Header name</Label>
            <Input
              id="factory-webhook-header"
              value={headerName}
              placeholder={headerDefault}
              onChange={(event) => {
                setError("");
                setHeaderName(event.target.value);
              }}
              data-testid="factory-webhook-header"
            />
          </div>
        ) : null}
        {error ? (
          <p className="text-[12px] text-destructive" role="alert">
            {error}
          </p>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <LoadingButton type="submit" size="sm" loading={pending} data-testid="factory-webhook-submit">
            {webhook ? "Save webhook" : "Create webhook"}
          </LoadingButton>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}

export function WebhookAuthenticationField({
  value,
  onChange,
  disabled,
}: {
  value: Authentication;
  onChange: (value: Authentication) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="factory-webhook-authentication">Authentication</Label>
      <Select value={value} onValueChange={(next) => onChange(next as Authentication)} disabled={disabled}>
        <SelectTrigger
          id="factory-webhook-authentication"
          className="w-full"
          data-testid="factory-webhook-authentication"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {AUTHENTICATION_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className="text-[12px] text-muted-foreground">{authenticationHelp(value)}</p>
    </div>
  );
}

export function WebhookSecretDialog({ reveal, onClose }: { reveal: Reveal | null; onClose: () => void }) {
  const webhook = reveal?.webhook;
  const mode = AUTHENTICATION_OPTIONS.find((option) => option.value === webhook?.authentication)?.mode ?? "bearer";
  const example = generateWebhookCode(mode, {
    webhookUrl: webhook?.url ?? "",
    headerName: webhook?.headerName || DEFAULT_WEBHOOK_HEADER_TOKEN_NAME,
    signatureHeaderName: webhook?.headerName || DEFAULT_WEBHOOK_SIGNATURE_HEADER,
    secret: reveal?.secret,
  });
  return (
    <Dialog open={Boolean(reveal)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg sm:max-w-2xl" data-testid="factory-webhook-secret-dialog">
        <DialogHeader>
          <DialogTitle>Copy the webhook secret</DialogTitle>
          <DialogDescription>Copy the secret now. You cannot see it again.</DialogDescription>
        </DialogHeader>
        {reveal ? (
          <div className="min-w-0 space-y-3">
            <CopyField label="Webhook URL" value={webhook?.url ?? ""} testId="factory-webhook-secret-url" />
            <CopyField label="Secret" value={reveal.secret} testId="factory-webhook-secret" />
            <div>
              <p className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
                Example request
              </p>
              <div className="relative w-full overflow-hidden rounded-md border border-border bg-muted/40">
                <div className="absolute right-2 top-2 z-10">
                  <CopyButton
                    text={example.code}
                    ariaLabel="Copy example request"
                    copiedAriaLabel="Example request copied"
                  />
                </div>
                <pre className="overflow-x-auto px-4 py-3 pr-12 text-[12px] leading-relaxed text-foreground">
                  {example.code}
                </pre>
              </div>
            </div>
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" size="sm" onClick={onClose} data-testid="factory-webhook-secret-done">
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CopyField({ label, value, testId }: { label: string; value: string; testId: string }) {
  return (
    <div>
      <p className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">{label}</p>
      <div className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2">
        <code className="min-w-0 flex-1 truncate text-left text-[12px] text-foreground" data-testid={testId}>
          {value}
        </code>
        <CopyButton text={value} ariaLabel={`Copy ${label.toLowerCase()}`} copiedAriaLabel={`${label} copied`} />
      </div>
    </div>
  );
}

function isAuthentication(value?: FactoryWebhookAuthentication): value is Authentication {
  return AUTHENTICATION_OPTIONS.some((option) => option.value === value);
}

function authenticationSummary(webhook: FactoriesFactoryWebhook): string {
  const label =
    AUTHENTICATION_OPTIONS.find((option) => option.value === webhook.authentication)?.label ?? "Bearer token";
  return webhook.headerName ? `${label} in ${webhook.headerName}` : label;
}

function authenticationHelp(authentication: Authentication): string {
  switch (authentication) {
    case "AUTHENTICATION_SIGNATURE":
      return "The sender signs the request body with the secret (HMAC SHA-256).";
    case "AUTHENTICATION_HEADER_TOKEN":
      return "The sender puts the secret in a custom header.";
    default:
      return "The sender puts the secret in the Authorization header.";
  }
}

function usageLabel(count: number): string {
  if (count === 0) {
    return "Not used by an automation";
  }
  return count === 1 ? "Used by 1 automation" : `Used by ${count} automations`;
}
