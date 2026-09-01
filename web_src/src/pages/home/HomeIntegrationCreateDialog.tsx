import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";
import { IntegrationCreateDialog, type IntegrationCreatePayload } from "@/ui/IntegrationCreateDialog";
import type { MutableRefObject } from "react";

import type { IntegrationSelections } from "./homeIntegrationStatus";
import { resolveDefaultDialogName } from "./integrationConnectDialogState";

export function HomeIntegrationCreateDialog({
  open,
  dialogIntegrationName,
  dialogMode,
  organizationId,
  integrationHomeHref,
  dialogDefinition,
  defaultDialogName,
  existingIntegrationNames,
  resumePendingForDialog,
  initialWebhookSetup,
  createIntegrationMutation,
  pendingConnectKeyRef,
  selections,
  onSelectionsChange,
  onPreferInstance,
  onClose,
  onCapabilitySetup,
  onRefetch,
  setupReturnTo,
  preferredCreateName,
  hiddenFieldNames,
}: {
  open: boolean;
  dialogIntegrationName: string | null;
  dialogMode: "create" | "resume";
  organizationId: string;
  integrationHomeHref?: string;
  setupReturnTo?: string;
  preferredCreateName?: string;
  /** Configuration field names this flow never shows for the open integration. */
  hiddenFieldNames?: string[];
  dialogDefinition: unknown;
  defaultDialogName: string;
  existingIntegrationNames: Set<string>;
  resumePendingForDialog?: OrganizationsIntegration;
  initialWebhookSetup?: { id: string; webhookUrl: string; config: Record<string, unknown> };
  createIntegrationMutation: {
    mutateAsync: (payload: IntegrationCreatePayload) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
    reset: () => void;
  };
  pendingConnectKeyRef: MutableRefObject<string | null>;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onPreferInstance: (integrationName: string, integrationId: string) => void;
  onClose: () => void;
  onCapabilitySetup: (integrationName: string, integrationId?: string) => void;
  onRefetch: () => void;
}) {
  return (
    <IntegrationCreateDialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          onClose();
          createIntegrationMutation.reset();
        }
      }}
      integrationDefinition={(dialogDefinition as never) ?? null}
      organizationId={organizationId}
      integrationHomeHref={integrationHomeHref}
      onCreateIntegration={async (payload) => {
        const res = await createIntegrationMutation.mutateAsync(payload);
        return res.data;
      }}
      onReset={() => createIntegrationMutation.reset()}
      defaultName={
        dialogMode === "create"
          ? resolveDefaultDialogName(dialogIntegrationName, undefined, existingIntegrationNames, preferredCreateName)
          : defaultDialogName
      }
      onCreated={(integrationId, instanceName) => {
        // Dialog calls onOpenChange(false) before onCreated; keep the key in a ref across that close.
        const key = pendingConnectKeyRef.current;
        pendingConnectKeyRef.current = null;
        if (key) {
          onPreferInstance(key, integrationId);
          // Newly created instances are not ready until setup finishes.
          onSelectionsChange({
            ...selections,
            [key]: { id: integrationId, name: instanceName, ready: false },
          });
        }
        onClose();
        onRefetch();
      }}
      onCapabilitySetupRequired={(integrationName, integrationId) => {
        pendingConnectKeyRef.current = null;
        onPreferInstance(integrationName, integrationId);
        onClose();
        onCapabilitySetup(integrationName, integrationId);
      }}
      initialBrowserAction={resumePendingForDialog?.status?.browserAction}
      initialCreatedIntegrationId={resumePendingForDialog?.metadata?.id}
      initialWebhookSetup={dialogMode === "create" ? undefined : initialWebhookSetup}
      initialConfiguration={resumePendingForDialog?.spec?.configuration as Record<string, unknown> | undefined}
      setupReturnTo={setupReturnTo}
      existingIntegrationNames={existingIntegrationNames}
      hiddenFieldNames={hiddenFieldNames}
    />
  );
}
