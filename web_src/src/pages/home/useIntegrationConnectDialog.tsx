import type { OrganizationsIntegration } from "@/api-client";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { useMemo, useRef, useState } from "react";

import { HomeIntegrationCreateDialog } from "./HomeIntegrationCreateDialog";
import {
  selectionFromInstance,
  type IntegrationInstanceSummary,
  type IntegrationSelections,
} from "./homeIntegrationStatus";
import { resolveIntegrationHomeHref, useCreateDialogProps } from "./integrationConnectDialogState";
import { useHomeIntegrationConnectActions } from "./useHomeIntegrationConnectActions";
import { useInstallIntegrationSelections, useRefetchOnWindowFocus } from "./useInstallIntegrationSelections";

function selectReadyIntegrationInstance(
  connected: OrganizationsIntegration[],
  selections: IntegrationSelections,
  integrationName: string,
  integrationId: string,
): IntegrationSelections | null {
  const instance = connected?.find(
    (item) => item.metadata?.integrationName === integrationName && item.metadata?.id === integrationId,
  );
  const selection = instance ? selectionFromInstance(instance) : null;
  return selection?.ready ? { ...selections, [integrationName]: selection } : null;
}

/**
 * Connect and configure flows for a set of integration types, without any row
 * UI. Callers render their own rows and mount `dialogs`, so the home installer
 * and workspace setup share one connect path.
 */
export function useIntegrationConnectDialog({
  organizationId,
  returnTo,
  integrationNames,
  selections,
  onSelectionsChange,
  preferredCreateNames,
}: {
  organizationId: string;
  returnTo?: string;
  integrationNames: string[];
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** When set, a new connection uses this name instead of the integration type name. */
  preferredCreateNames?: Record<string, string>;
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  /** "create" skips resuming a pending instance so "Connect new" always starts fresh. */
  const [dialogMode, setDialogMode] = useState<"create" | "resume">("resume");
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  const pendingConnectKeyRef = useRef<string | null>(null);

  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );
  const integrationData: IntegrationInstanceSummary[] = useMemo(
    () =>
      integrationNames.map((name) => {
        const allInstances = connected.filter((item) => item.metadata?.integrationName === name);
        return { name, allInstances, readyInstances: allInstances.filter((item) => item.status?.state === "ready") };
      }),
    [integrationNames, connected],
  );
  const { rememberPreferredInstance } = useInstallIntegrationSelections({
    integrationData,
    selections,
    onSelectionsChange,
  });
  useRefetchOnWindowFocus(refetch);

  const preferredCreateName = dialogIntegrationName ? preferredCreateNames?.[dialogIntegrationName] : undefined;
  const { dialogDefinition, dialogPendingInstance, initialWebhookSetup, defaultDialogName } = useCreateDialogProps(
    dialogIntegrationName,
    availableIntegrations,
    connected,
    existingIntegrationNames,
    preferredCreateName,
  );
  const integrationHomeHref = useMemo(
    () =>
      resolveIntegrationHomeHref({
        organizationId,
        dialogIntegrationName,
        dialogMode,
        pendingId: dialogPendingInstance?.metadata?.id,
        selectedId: dialogIntegrationName ? selections[dialogIntegrationName]?.id : undefined,
      }),
    [organizationId, dialogIntegrationName, dialogMode, dialogPendingInstance?.metadata?.id, selections],
  );
  const { openCapabilitySetup, openCreateIntegrationModal, openConnectDialog, openConfigureDialog } =
    useHomeIntegrationConnectActions({
      organizationId,
      availableIntegrations,
      connected,
      pendingConnectKeyRef,
      setDialogMode,
      setDialogIntegrationName,
      setConfigureIntegrationId,
    });

  const selectInstance = (integrationName: string, integrationId: string) => {
    const next = selectReadyIntegrationInstance(connected, selections, integrationName, integrationId);
    if (next) onSelectionsChange(next);
  };

  const dialogs = (
    <>
      <ConfigureIntegrationDialog
        integrationId={configureIntegrationId}
        organizationId={organizationId}
        onClose={() => {
          setConfigureIntegrationId(null);
          void refetch();
        }}
      />
      <HomeIntegrationCreateDialog
        open={!!dialogIntegrationName}
        dialogIntegrationName={dialogIntegrationName}
        dialogMode={dialogMode}
        organizationId={organizationId}
        integrationHomeHref={integrationHomeHref}
        dialogDefinition={dialogDefinition}
        defaultDialogName={defaultDialogName}
        existingIntegrationNames={existingIntegrationNames}
        resumePendingForDialog={dialogMode === "resume" ? dialogPendingInstance : undefined}
        initialWebhookSetup={initialWebhookSetup}
        createIntegrationMutation={createIntegrationMutation}
        pendingConnectKeyRef={pendingConnectKeyRef}
        selections={selections}
        onSelectionsChange={onSelectionsChange}
        onPreferInstance={rememberPreferredInstance}
        onClose={() => {
          setDialogIntegrationName(null);
          setDialogMode("resume");
        }}
        onCapabilitySetup={(integrationName, integrationId) => {
          if (integrationId) rememberPreferredInstance(integrationName, integrationId);
          openCapabilitySetup(integrationName, integrationId);
          void refetch();
        }}
        onRefetch={() => void refetch()}
        setupReturnTo={returnTo}
        preferredCreateName={preferredCreateName}
      />
    </>
  );

  return {
    integrationData,
    requestConnect: openConnectDialog,
    createNew: openCreateIntegrationModal,
    selectInstance,
    configure: openConfigureDialog,
    dialogs,
  };
}
