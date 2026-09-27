import { type RefObject, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import type {
  IntegrationsIntegrationDefinition,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";
import {
  integrationKeys,
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
} from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { useSyncGitHubConnection } from "@/hooks/useSyncGitHubConnection";
import { getApiErrorMessage } from "@/lib/errors";
import { peekIntegrationSetupReturnPreferredIntegration } from "@/lib/integrationSetupReturn";
import {
  offersPrivateGitHubAppSetup,
  usesHostedGitHubAppInstall,
  usesHostedJiraOAuth,
  usesPrivateGitHubAppWizard,
} from "@/lib/integrations";
import { connectPrivateGitHubApp } from "@/lib/privateGitHubApp";
import {
  hostedGitHubConnectUserGate,
  persistGitHubSetupReturnPath,
  startDirectGitHubConnect,
} from "@/lib/startDirectGitHubConnect";
import { startDirectJiraConnect } from "@/lib/startDirectJiraConnect";
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

export function selectReadyIntegrationInstance(
  connected: OrganizationsIntegration[],
  selections: IntegrationSelections,
  integrationName: string,
  integrationId?: string,
): IntegrationSelections | null {
  const instance = connected?.find(
    (item) =>
      item.metadata?.integrationName === integrationName &&
      (!integrationId || item.metadata?.id === integrationId) &&
      item.status?.state === "ready",
  );
  const selection = instance ? selectionFromInstance(instance) : null;
  return selection?.ready ? { ...selections, [integrationName]: selection } : null;
}

/** Prefer the query cache after an async mutation refreshes a connection. */
export function selectLatestReadyIntegrationInstance(
  refreshed: OrganizationsIntegration[] | undefined,
  rendered: OrganizationsIntegration[],
  selections: IntegrationSelections,
  integrationName: string,
  integrationId?: string,
): IntegrationSelections | null {
  return selectReadyIntegrationInstance(refreshed ?? rendered, selections, integrationName, integrationId);
}

function useLatestReadyIntegrationSelection(args: {
  organizationId: string;
  connected: OrganizationsIntegration[];
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
}) {
  const queryClient = useQueryClient();
  const selections = useRef(args.selections);
  selections.current = args.selections;

  return (integrationName: string, integrationId: string): boolean => {
    const refreshed = queryClient.getQueryData<OrganizationsIntegration[]>(
      integrationKeys.connected(args.organizationId),
    );
    const next = selectLatestReadyIntegrationInstance(
      refreshed,
      args.connected,
      selections.current,
      integrationName,
      integrationId,
    );
    if (!next) return false;
    args.onSelectionsChange(next);
    return true;
  };
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
  manualSelectionNames,
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
  /** Integration names that are never auto-selected; the user must pick an instance. */
  manualSelectionNames?: readonly string[];
}) {
  const { data: me, isError: meFailed, isSuccess: meLoaded } = useMe(true, organizationId);
  const {
    data: connected = [],
    refetch,
    isLoading: connectionsLoading,
  } = useConnectedIntegrations(organizationId, {
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
    manualSelectionNames,
    loading: connectionsLoading,
    initialPreferredIntegrationId: peekIntegrationSetupReturnPreferredIntegration(organizationId),
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

  const hostedConnect = useHostedProviderConnect({
    organizationId,
    returnTo,
    connected,
    existingIntegrationNames,
    currentUserId: me?.id,
    currentUserResolved: meLoaded || meFailed,
    createIntegration: createIntegrationMutation.mutateAsync,
  });

  const requestConnect = async (integrationName: string, preferredIntegrationId?: string): Promise<boolean> => {
    if (integrationName === "github" && githubConnect.hosted) {
      return hostedConnect.github(false, preferredIntegrationId);
    }
    const existingSelection = selectReadyIntegrationInstance(connected, selections, integrationName);
    if (existingSelection) {
      onSelectionsChange(existingSelection);
      // Selecting an in-memory connection does not navigate away. Callers
      // must be able to finish their current action and render the selection.
      return false;
    }
    if (integrationName === "jira" && isHostedJira(availableIntegrations)) {
      return hostedConnect.jira();
    }
    openConnectDialog(integrationName);
    return false;
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
      void hostedConnect.github(true);
      return;
    }
    if (integrationName === "jira" && isHostedJira(availableIntegrations)) {
      void hostedConnect.jira(true);
      return;
    }
    openCreateIntegrationModal(integrationName);
  };

  const selectInstance = useLatestReadyIntegrationSelection({
    organizationId,
    connected,
    selections,
    onSelectionsChange,
  });

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
    /** True while the first load of the connected list is in flight. */
    connectionsLoading,
    /** Refetches the connected list, for screens that must not show a stale cache. */
    refetchConnections: refetch,
    /** Synchronizes one hosted GitHub connection and updates the connected-list cache. */
    syncGitHubConnection: hostedConnect.syncGitHubConnection,
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

function isHostedJira(availableIntegrations: IntegrationsIntegrationDefinition[]) {
  return usesHostedJiraOAuth(availableIntegrations.find((item) => item.name === "jira"));
}

type QueuedHostedGitHubConnect = {
  forceNew: boolean;
  preferredIntegrationId?: string;
  result: Promise<boolean>;
  resolve: (navigationStarted: boolean) => void;
};

function queueHostedGitHubConnect(
  pending: RefObject<QueuedHostedGitHubConnect | null>,
  forceNew: boolean,
  preferredIntegrationId?: string,
): Promise<boolean> {
  if (pending.current) return pending.current.result;

  let resolve!: (navigationStarted: boolean) => void;
  const result = new Promise<boolean>((promiseResolve) => {
    resolve = promiseResolve;
  });
  pending.current = { forceNew, preferredIntegrationId, result, resolve };
  return result;
}

export function useHostedGitHubConnect({
  organizationId,
  returnTo,
  connected,
  existingIntegrationNames,
  currentUserId,
  currentUserResolved,
  createIntegration,
}: {
  organizationId: string;
  returnTo?: string;
  connected: OrganizationsIntegration[];
  existingIntegrationNames: Set<string>;
  currentUserId?: string;
  currentUserResolved: boolean;
  createIntegration: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
}) {
  const navigate = useNavigate();
  const pendingGitHubConnectRef = useRef<QueuedHostedGitHubConnect | null>(null);

  const connectGitHubWithoutDialog = useCallback(
    async (forceNew = false, preferredIntegrationId?: string): Promise<boolean> => {
      const userGate = hostedGitHubConnectUserGate(currentUserId, currentUserResolved);
      if (userGate !== "run") {
        if (userGate === "queue") {
          return queueHostedGitHubConnect(pendingGitHubConnectRef, forceNew, preferredIntegrationId);
        } else {
          pendingGitHubConnectRef.current = null;
          showErrorToast("Failed to connect GitHub");
        }
        return false;
      }

      pendingGitHubConnectRef.current = null;
      try {
        return await startDirectGitHubConnect({
          organizationId,
          returnTo,
          existingNames: existingIntegrationNames,
          connected,
          currentUserId,
          forceNew,
          preferredIntegrationId,
          goTo: navigate,
          create: async (payload) => {
            const response = await createIntegration(payload);
            return response.data;
          },
          update: persistGitHubSetupReturnPath(organizationId),
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to connect GitHub"));
        return false;
      }
    },
    [
      connected,
      createIntegration,
      currentUserId,
      currentUserResolved,
      existingIntegrationNames,
      navigate,
      organizationId,
      returnTo,
    ],
  );

  useEffect(() => {
    const pending = pendingGitHubConnectRef.current;
    if (!pending) return;
    if (hostedGitHubConnectUserGate(currentUserId, currentUserResolved) === "queue") {
      return;
    }

    pendingGitHubConnectRef.current = null;
    void connectGitHubWithoutDialog(pending.forceNew, pending.preferredIntegrationId).then(pending.resolve);
  }, [connectGitHubWithoutDialog, currentUserId, currentUserResolved]);

  return connectGitHubWithoutDialog;
}

function useHostedProviderConnect({
  organizationId,
  returnTo,
  connected,
  existingIntegrationNames,
  currentUserId,
  currentUserResolved,
  createIntegration,
}: {
  organizationId: string;
  returnTo?: string;
  connected: OrganizationsIntegration[];
  existingIntegrationNames: Set<string>;
  currentUserId?: string;
  currentUserResolved: boolean;
  createIntegration: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
}) {
  return {
    syncGitHubConnection: useSyncGitHubConnection(organizationId),
    github: useHostedGitHubConnect({
      organizationId,
      returnTo,
      connected,
      existingIntegrationNames,
      currentUserId,
      currentUserResolved,
      createIntegration,
    }),
    jira: useHostedJiraConnect({
      organizationId,
      returnTo,
      connected,
      existingIntegrationNames,
      createIntegration,
    }),
  };
}

export function useHostedJiraConnect({
  organizationId,
  returnTo,
  connected,
  existingIntegrationNames,
  createIntegration,
}: {
  organizationId: string;
  returnTo?: string;
  connected: OrganizationsIntegration[];
  existingIntegrationNames: Set<string>;
  createIntegration: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
}) {
  return useCallback(
    async (forceNew = false): Promise<boolean> => {
      try {
        return await startDirectJiraConnect({
          organizationId,
          returnTo,
          existingNames: existingIntegrationNames,
          connected,
          forceNew,
          create: async (payload) => {
            const response = await createIntegration(payload);
            return response.data;
          },
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to connect Jira"));
        return false;
      }
    },
    [connected, createIntegration, existingIntegrationNames, organizationId, returnTo],
  );
}
