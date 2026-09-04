import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import type {
  IntegrationsIntegrationDefinition,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { getApiErrorMessage } from "@/lib/errors";
import {
  offersPrivateGitHubAppSetup,
  usesHostedGitHubAppInstall,
  usesPrivateGitHubAppWizard,
} from "@/lib/integrations";
import { connectPrivateGitHubApp } from "@/lib/privateGitHubApp";
import { persistGitHubSetupReturnPath, startDirectGitHubConnect } from "@/lib/startDirectGitHubConnect";
import { showErrorToast } from "@/lib/toast";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";

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
  hiddenConfigurationFields,
}: {
  organizationId: string;
  returnTo?: string;
  integrationNames: string[];
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** When set, a new connection uses this name instead of the integration type name. */
  preferredCreateNames?: Record<string, string>;
  /** Configuration field names the create dialog never shows, keyed by integration name. */
  hiddenConfigurationFields?: Record<string, string[]>;
}) {
  const { data: me } = useMe(true, organizationId);
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations({
    enabled: !!organizationId,
    organizationId,
  });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const navigate = useNavigate();
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
  const githubConnect = githubConnectFlags(availableIntegrations);
  const { openCapabilitySetup, openCreateIntegrationModal, openConnectDialog, openConfigureDialog } =
    useHomeIntegrationConnectActions({
      organizationId,
      returnTo,
      availableIntegrations,
      connected,
      pendingConnectKeyRef,
      setDialogMode,
      setDialogIntegrationName,
      setConfigureIntegrationId,
    });

  const connectGitHubWithoutDialog = useHostedGitHubConnect({
    organizationId,
    returnTo,
    connected,
    existingIntegrationNames,
    currentUserId: me?.id,
    createIntegration: createIntegrationMutation.mutateAsync,
  });

  const requestConnect = (integrationName: string) => {
    if (integrationName === "github" && githubConnect.hosted) {
      void connectGitHubWithoutDialog();
      return;
    }
    openConnectDialog(integrationName);
  };

  const requestPrivateGitHubConnect = useCallback(() => {
    void connectPrivateGitHubApp({
      useWizard: githubConnect.useWizard,
      organizationId,
      returnTo,
      existingNames: existingIntegrationNames,
      connected,
      currentUserId: me?.id,
      goTo: navigate,
      create: async (payload) => {
        const response = await createIntegrationMutation.mutateAsync(payload);
        return response.data;
      },
    }).catch((error) => {
      showErrorToast(getApiErrorMessage(error, "Failed to connect GitHub"));
    });
  }, [
    connected,
    createIntegrationMutation,
    existingIntegrationNames,
    githubConnect.useWizard,
    me?.id,
    navigate,
    organizationId,
    returnTo,
  ]);

  const createNew = (integrationName: string) => {
    if (integrationName === "github" && githubConnect.hosted) {
      void connectGitHubWithoutDialog(true);
      return;
    }
    openCreateIntegrationModal(integrationName);
  };

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
        hiddenFieldNames={dialogIntegrationName ? hiddenConfigurationFields?.[dialogIntegrationName] : undefined}
      />
    </>
  );

  return {
    integrationData,
    requestConnect,
    requestPrivateGitHubConnect,
    hostedGitHubAppInstall: githubConnect.hosted,
    offersPrivateGitHubAppSetup: githubConnect.privateApp,
    createNew,
    selectInstance,
    configure: openConfigureDialog,
    dialogs,
  };
}

function githubConnectFlags(availableIntegrations: IntegrationsIntegrationDefinition[]) {
  const githubDefinition = availableIntegrations.find((item) => item.name === "github");
  return {
    hosted: usesHostedGitHubAppInstall(githubDefinition),
    privateApp: offersPrivateGitHubAppSetup(githubDefinition),
    useWizard: usesPrivateGitHubAppWizard(githubDefinition),
  };
}

function useHostedGitHubConnect({
  organizationId,
  returnTo,
  connected,
  existingIntegrationNames,
  currentUserId,
  createIntegration,
}: {
  organizationId: string;
  returnTo?: string;
  connected: OrganizationsIntegration[];
  existingIntegrationNames: Set<string>;
  currentUserId?: string;
  createIntegration: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
}) {
  const navigate = useNavigate();
  const pendingGitHubConnectRef = useRef<false | { forceNew: boolean }>(false);

  const connectGitHubWithoutDialog = useCallback(
    async (forceNew = false) => {
      if (!currentUserId) {
        pendingGitHubConnectRef.current = { forceNew };
        return;
      }

      pendingGitHubConnectRef.current = false;
      try {
        await startDirectGitHubConnect({
          organizationId,
          returnTo,
          existingNames: existingIntegrationNames,
          connected,
          currentUserId,
          forceNew,
          goTo: navigate,
          create: async (payload) => {
            const response = await createIntegration(payload);
            return response.data;
          },
          update: persistGitHubSetupReturnPath(organizationId),
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to connect GitHub"));
      }
    },
    [connected, createIntegration, currentUserId, existingIntegrationNames, navigate, organizationId, returnTo],
  );

  useEffect(() => {
    if (!currentUserId || !pendingGitHubConnectRef.current) {
      return;
    }

    const { forceNew } = pendingGitHubConnectRef.current;
    pendingGitHubConnectRef.current = false;
    void connectGitHubWithoutDialog(forceNew);
  }, [connectGitHubWithoutDialog, currentUserId]);

  return connectGitHubWithoutDialog;
}
