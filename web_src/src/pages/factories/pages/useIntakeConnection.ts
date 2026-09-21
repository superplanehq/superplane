import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { followBrowserAction } from "@/lib/browserAction";
import { getApiErrorMessage } from "@/lib/errors";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { usesHostedJiraOAuth } from "@/lib/integrations";
import { startDirectJiraConnect } from "@/lib/startDirectJiraConnect";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  filterIntakeConnections,
  INTAKE_CONNECTION_COPY,
  intakeConnectionReturnPath,
  intakeProviderAppName,
  intakeSourceAllowsRebind,
  type IntakeConnectionBinding,
} from "./intakeConnectionModel";
import { isHostedSentryInstallAction } from "./useSentryIntakeSetup";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function useIntakeConnection(args: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  intakeId: string;
  sourceId: LineIntakeSourceId;
  initialBinding: IntakeConnectionBinding;
  returnedIntegrationId?: string;
}) {
  const enabled = intakeSourceAllowsRebind(args.sourceId);
  const [binding, setBinding] = useIntakeBinding(args.initialBinding);
  const [connectOpen, setConnectOpen] = useState(false);
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState<string>();
  const returnPath = intakeConnectionReturnPath(args.organizationId, args.factoryKey, args.lineId, args.intakeId);
  const queries = useIntakeConnectionQueries(args.organizationId, args.sourceId, binding.integrationId, enabled);

  useReturnedIntakeConnection({
    enabled,
    returnedIntegrationId: args.returnedIntegrationId,
    integrations: queries.integrations,
    refetch: queries.connectedQuery.refetch,
    setBinding,
    setConnectOpen,
  });

  const completeConnection = (connectedIntegrationId: string) => {
    setBinding({ integrationId: connectedIntegrationId, resourceId: "" });
    setConnectOpen(false);
    void queries.connectedQuery.refetch();
  };

  const connect = () =>
    connectProvider({
      enabled,
      sourceId: args.sourceId,
      organizationId: args.organizationId,
      returnPath,
      connected: queries.connectedQuery.data ?? [],
      existingNames: queries.existingNames,
      definition: queries.definition,
      definitionLoading: queries.availableQuery.isLoading,
      createIntegration: queries.createIntegration,
      completeConnection,
      setBinding,
      setConnectOpen,
      setConnecting,
      setError,
    });

  return {
    enabled,
    binding,
    setBinding,
    integrations: queries.integrations,
    integrationsLoading: queries.connectedQuery.isLoading,
    projects: queries.projectsQuery.data ?? [],
    projectsLoading: queries.projectsQuery.isLoading,
    projectsError: queries.projectsQuery.isError,
    retryProjects: () => void queries.projectsQuery.refetch(),
    connecting,
    connectError: error,
    connectOpen,
    setConnectOpen,
    configureIntegrationId,
    closeConfigure: () => setConfigureIntegrationId(null),
    definition: queries.definition,
    existingNames: queries.existingNames,
    createIntegration: queries.createIntegration,
    completeConnection,
    connect,
    reconnect: () => {
      if (binding.integrationId) {
        setConfigureIntegrationId(binding.integrationId);
      }
    },
    returnPath,
  };
}

function useIntakeBinding(initialBinding: IntakeConnectionBinding) {
  const [binding, setBinding] = useState(() => initialBinding);
  useEffect(() => {
    setBinding({
      integrationId: initialBinding.integrationId,
      resourceId: initialBinding.resourceId,
    });
  }, [initialBinding.integrationId, initialBinding.resourceId]);
  return [binding, setBinding] as const;
}

function useIntakeConnectionQueries(
  organizationId: string,
  sourceId: LineIntakeSourceId,
  integrationId: string,
  enabled: boolean,
) {
  const connectedQuery = useConnectedIntegrations(organizationId, { enabled });
  const availableQuery = useAvailableIntegrations({ organizationId, enabled });
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: enabled && Boolean(integrationId),
  });
  const integrations = useMemo(
    () => (enabled ? filterIntakeConnections(connectedQuery.data ?? [], sourceId) : []),
    [enabled, connectedQuery.data, sourceId],
  );
  const definition = useMemo(
    () => availableQuery.data?.find((item) => item.name === intakeProviderAppName(sourceId)),
    [availableQuery.data, sourceId],
  );
  const existingNames = useMemo(
    () =>
      new Set(
        (connectedQuery.data ?? [])
          .map((integration) => integration.metadata?.name?.trim())
          .filter((name): name is string => Boolean(name)),
      ),
    [connectedQuery.data],
  );
  return { connectedQuery, availableQuery, createIntegration, projectsQuery, integrations, definition, existingNames };
}

function useReturnedIntakeConnection({
  enabled,
  returnedIntegrationId,
  integrations,
  refetch,
  setBinding,
  setConnectOpen,
}: {
  enabled: boolean;
  returnedIntegrationId?: string;
  integrations: OrganizationsIntegration[];
  refetch: () => Promise<unknown>;
  setBinding: (next: IntakeConnectionBinding | ((current: IntakeConnectionBinding) => IntakeConnectionBinding)) => void;
  setConnectOpen: (open: boolean) => void;
}) {
  const pickedReturnedConnection = useRef(false);

  useEffect(() => {
    pickedReturnedConnection.current = false;
  }, [returnedIntegrationId]);

  useEffect(() => {
    if (!enabled || !returnedIntegrationId || pickedReturnedConnection.current) {
      return;
    }
    const returned = integrations.find((integration) => integration.metadata?.id === returnedIntegrationId);
    if (!returned?.metadata?.id) {
      return;
    }
    const returnedId = returned.metadata.id;
    pickedReturnedConnection.current = true;
    setBinding((current) =>
      current.integrationId === returnedId ? current : { integrationId: returnedId, resourceId: "" },
    );
    setConnectOpen(false);
    void refetch();
  }, [enabled, returnedIntegrationId, integrations, refetch, setBinding, setConnectOpen]);
}

type ConnectArgs = {
  organizationId: string;
  returnPath: string;
  connected: OrganizationsIntegration[];
  existingNames: Set<string>;
  createIntegration: ReturnType<typeof useCreateIntegration>;
  setConnectOpen: (open: boolean) => void;
  setConnecting: (connecting: boolean) => void;
  setError: (message?: string) => void;
};

async function connectProvider(
  args: ConnectArgs & {
    enabled: boolean;
    sourceId: LineIntakeSourceId;
    definition?: IntegrationsIntegrationDefinition;
    definitionLoading: boolean;
    completeConnection: (integrationId: string) => void;
    setBinding: (next: IntakeConnectionBinding) => void;
  },
) {
  if (!args.enabled) {
    return;
  }
  args.setError(undefined);
  if (args.sourceId === "jira-issues") {
    await connectJira(args);
    return;
  }
  if (args.sourceId === "sentry-exceptions") {
    await connectSentry({
      ...args,
      adoptConnection: (integrationId: string) => {
        args.setBinding({ integrationId, resourceId: "" });
      },
    });
    return;
  }
  args.setConnectOpen(true);
}

async function connectJira(
  args: ConnectArgs & {
    definition?: IntegrationsIntegrationDefinition;
    definitionLoading: boolean;
    completeConnection: (integrationId: string) => void;
  },
) {
  if (args.definitionLoading) {
    return;
  }
  if (!usesHostedJiraOAuth(args.definition)) {
    args.setConnectOpen(true);
    return;
  }

  args.setConnecting(true);
  try {
    await startDirectJiraConnect({
      organizationId: args.organizationId,
      returnTo: args.returnPath,
      existingNames: args.existingNames,
      connected: args.connected,
      onExistingReady: args.completeConnection,
      create: async (payload) => {
        const response = await args.createIntegration.mutateAsync(payload);
        return response.data;
      },
    });
  } catch (cause) {
    args.setError(getApiErrorMessage(cause, INTAKE_CONNECTION_COPY.connectError));
  } finally {
    args.setConnecting(false);
  }
}

async function connectSentry(
  args: ConnectArgs & {
    completeConnection: (integrationId: string) => void;
    adoptConnection: (integrationId: string) => void;
  },
) {
  args.setConnecting(true);
  try {
    const { result } = await createWithGeneratedName({
      baseName: "sentry",
      takenNames: args.existingNames,
      create: (name) =>
        args.createIntegration.mutateAsync({
          integrationName: "sentry",
          name,
          configuration: { setupReturnPath: args.returnPath },
        }),
    });
    const integration = result.data?.integration;
    if (integration?.status?.state === "ready" && integration.metadata?.id) {
      args.completeConnection(integration.metadata.id);
      return;
    }

    rememberIntegrationSetupReturn(args.organizationId, args.returnPath);
    const action = integration?.status?.browserAction;
    if (isHostedSentryInstallAction(action?.url)) {
      followBrowserAction(action);
      return;
    }
    if (integration?.metadata?.id) {
      args.adoptConnection(integration.metadata.id);
    }
    args.setConnectOpen(true);
  } catch (cause) {
    args.setError(getApiErrorMessage(cause, INTAKE_CONNECTION_COPY.connectError));
  } finally {
    args.setConnecting(false);
  }
}

export function intakeConnectionDefaultName(sourceId: LineIntakeSourceId): string {
  if (sourceId === "jira-issues") {
    return "Jira";
  }
  if (sourceId === "sentry-exceptions") {
    return "Sentry";
  }
  return "Productive";
}

export type IntakeConnectionModel = ReturnType<typeof useIntakeConnection>;
