import type { OrganizationsIntegration } from "@/api-client";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { useMemo } from "react";

export function resolveIntegrationHomeHref(args: {
  organizationId: string;
  dialogIntegrationName: string | null;
  dialogMode: "create" | "resume";
  pendingId?: string;
  selectedId?: string;
}) {
  if (!args.organizationId) return undefined;
  const integrationId = (args.dialogMode === "resume" ? args.pendingId : undefined) ?? args.selectedId;
  if (integrationId) return `/${args.organizationId}/settings/integrations/${integrationId}`;
  return `/${args.organizationId}/settings/integrations`;
}

function buildWebhookSetup(pending: OrganizationsIntegration | undefined) {
  const webhookUrl = getIntegrationWebhookUrl(pending?.status?.metadata);
  if (!webhookUrl || !pending?.metadata?.id) return undefined;
  return { id: pending.metadata.id, webhookUrl, config: { ...(pending.spec?.configuration ?? {}) } };
}

export function resolveDefaultDialogName(
  dialogIntegrationName: string | null,
  pending: OrganizationsIntegration | undefined,
  existingNames: Set<string>,
  preferredName?: string,
): string {
  if (pending?.metadata?.name) return pending.metadata.name;
  if (preferredName?.trim()) return getNextIntegrationName(preferredName.trim(), existingNames);
  if (!dialogIntegrationName) return "";
  return getNextIntegrationName(dialogIntegrationName, existingNames);
}

export function useCreateDialogProps(
  dialogIntegrationName: string | null,
  availableIntegrations: Array<{ name?: string; [key: string]: unknown }>,
  connected: OrganizationsIntegration[],
  existingIntegrationNames: Set<string>,
  preferredName?: string,
) {
  const dialogDefinition = useMemo(
    () => (dialogIntegrationName ? availableIntegrations.find((d) => d.name === dialogIntegrationName) : undefined),
    [availableIntegrations, dialogIntegrationName],
  );

  const dialogPendingInstance = useMemo(
    () =>
      dialogIntegrationName
        ? connected.find((i) => i.metadata?.integrationName === dialogIntegrationName && i.status?.state !== "ready")
        : undefined,
    [dialogIntegrationName, connected],
  );

  const initialWebhookSetup = useMemo(() => buildWebhookSetup(dialogPendingInstance), [dialogPendingInstance]);

  const defaultDialogName = useMemo(
    () =>
      resolveDefaultDialogName(dialogIntegrationName, dialogPendingInstance, existingIntegrationNames, preferredName),
    [dialogIntegrationName, dialogPendingInstance, existingIntegrationNames, preferredName],
  );

  return { dialogDefinition, dialogPendingInstance, initialWebhookSetup, defaultDialogName };
}
