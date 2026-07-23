import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { isCapabilityBasedIntegration, isCapabilityBasedIntegrationDefinition } from "@/lib/integrations";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

type ConnectDialogMode = "create" | "resume";

export function useHomeIntegrationConnectActions({
  organizationId,
  availableIntegrations,
  connected,
  pendingConnectKeyRef,
  setDialogMode,
  setDialogIntegrationName,
  setConfigureIntegrationId,
}: {
  organizationId: string;
  availableIntegrations: Array<{ name?: string }>;
  connected: OrganizationsIntegration[];
  pendingConnectKeyRef: MutableRefObject<string | null>;
  setDialogMode: Dispatch<SetStateAction<ConnectDialogMode>>;
  setDialogIntegrationName: Dispatch<SetStateAction<string | null>>;
  setConfigureIntegrationId: Dispatch<SetStateAction<string | null>>;
}) {
  const openCapabilitySetup = (integrationName: string, integrationId?: string) => {
    const path = `/${organizationId}/settings/integrations/${integrationName}/setup`;
    const href = integrationId ? `${path}?integrationId=${encodeURIComponent(integrationId)}` : path;
    // Keep factory setup on the current tab; finish GitHub install in a new one.
    window.open(href, "_blank", "noopener,noreferrer");
  };

  /** Always opens the in-page create modal (used by "Create new…" in the switcher). */
  const openCreateIntegrationModal = (integrationName: string) => {
    pendingConnectKeyRef.current = integrationName;
    setDialogMode("create");
    setDialogIntegrationName(integrationName);
  };

  /**
   * Two GitHub setup paths (see registry.SupportsNewSetupFlow):
   * - SetupProvider enabled (APP_ENV=development): definition.legacySetupOnly === false →
   *   multi-step wizard at /settings/integrations/:name/setup
   * - SetupProvider disabled / non-dev: legacySetupOnly === true →
   *   IntegrationCreateDialog + Sync browserAction (Continue on GitHub in the modal)
   */
  const openConnectDialog = (integrationName: string) => {
    const definition = availableIntegrations.find((item) => item.name === integrationName) as
      | IntegrationsIntegrationDefinition
      | undefined;
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
