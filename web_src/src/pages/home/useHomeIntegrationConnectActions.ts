import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import {
  isCapabilityBasedIntegration,
  isCapabilityBasedIntegrationDefinition,
  usesHostedGitHubAppInstall,
} from "@/lib/integrations";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

type ConnectDialogMode = "create" | "resume";

export function useHomeIntegrationConnectActions({
  organizationId,
  returnTo,
  availableIntegrations,
  connected,
  pendingConnectKeyRef,
  setDialogMode,
  setDialogIntegrationName,
  setConfigureIntegrationId,
}: {
  organizationId: string;
  returnTo?: string;
  availableIntegrations: IntegrationsIntegrationDefinition[];
  connected: OrganizationsIntegration[];
  pendingConnectKeyRef: MutableRefObject<string | null>;
  setDialogMode: Dispatch<SetStateAction<ConnectDialogMode>>;
  setDialogIntegrationName: Dispatch<SetStateAction<string | null>>;
  setConfigureIntegrationId: Dispatch<SetStateAction<string | null>>;
}) {
  const openCapabilitySetup = (integrationName: string, integrationId?: string) => {
    const path = `/${organizationId}/settings/integrations/${integrationName}/setup`;
    const href = integrationId ? `${path}?integrationId=${encodeURIComponent(integrationId)}` : path;
    rememberIntegrationSetupReturn(organizationId, returnTo);
    // Keep factory setup on the current tab; finish GitHub install in a new one.
    window.open(href, "_blank", "noopener,noreferrer");
  };

  /** Always opens the in-page create modal (used by "Connect new" in the switcher). */
  const openCreateIntegrationModal = (integrationName: string) => {
    pendingConnectKeyRef.current = integrationName;
    setDialogMode("create");
    setDialogIntegrationName(integrationName);
  };

  /**
   * Two GitHub setup paths (see registry.UseNewSetupFlow):
   * - hosted public app (definition.hostedAppInstall): skip the dialog and
   *   follow the Sync browserAction to the SuperPlane GitHub App install page
   * - org experimental feature new_integration_setup_flow + SetupProvider:
   *   definition.legacySetupOnly === false → wizard at /settings/integrations/:name/setup
   * - feature off: legacySetupOnly === true →
   *   IntegrationCreateDialog + Sync browserAction (Continue on GitHub in the modal)
   */
  const openConnectDialog = (integrationName: string) => {
    const definition = availableIntegrations.find((item) => item.name === integrationName);
    // Hosted Connect is started by requestConnect, not this dialog.
    if (usesHostedGitHubAppInstall(definition)) {
      return;
    }
    if (definition && isCapabilityBasedIntegrationDefinition(definition)) {
      const pending = connected.find(
        (item) => item.metadata?.integrationName === integrationName && item.status?.state !== "ready",
      );
      openCapabilitySetup(integrationName, pending?.metadata?.id);
      return;
    }

    pendingConnectKeyRef.current = integrationName;
    setDialogMode("resume");
    setDialogIntegrationName(integrationName);
  };

  const openConfigureDialog = (integrationId: string) => {
    const instance = connected.find((item) => item.metadata?.id === integrationId);
    const integrationName = instance?.metadata?.integrationName;
    // Prefer instance.legacySetup (actual create path) over the definition flag.
    if (
      integrationName &&
      instance &&
      isCapabilityBasedIntegration(instance) &&
      instance.status?.setupState?.currentStep
    ) {
      openCapabilitySetup(integrationName, integrationId);
      return;
    }
    setConfigureIntegrationId(integrationId);
  };

  return {
    openCapabilitySetup,
    openCreateIntegrationModal,
    openConnectDialog,
    openConfigureDialog,
  };
}
