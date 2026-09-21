import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { followBrowserAction } from "@/lib/browserAction";
import { getApiErrorMessage } from "@/lib/errors";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";
import { useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router";

export type SentrySetupStep = "connection" | "project";

export function useSentryIntakeSetup(organizationId: string, factoryId: string) {
  const [step, setStep] = useState<SentrySetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, sentryIntegrations, sentryDefinition, existingNames } = useSentryConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: Boolean(integrationId),
  });

  useEffect(() => {
    if (!integrationId && sentryIntegrations.length === 1) {
      setIntegrationId(sentryIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, sentryIntegrations]);

  useEffect(() => {
    if (stayOnConnection || step !== "connection") {
      return;
    }
    const readyId = readySentryConnectionId(sentryIntegrations, integrationId);
    if (!readyId) {
      return;
    }
    setIntegrationId(readyId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [stayOnConnection, step, sentryIntegrations, integrationId, connectedQuery]);

  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  };

  const { connecting, connectSentry } = useSentryConnect({
    organizationId,
    integrations: sentryIntegrations,
    integrationId,
    existingNames,
    createIntegration,
    completeConnection,
    setIntegrationId,
    setConnectOpen,
    setError,
  });

  const returnToConnection = () => {
    setStayOnConnection(true);
    setStep("connection");
  };

  const createBoundIntake = async () => {
    if (!integrationId || !projectId) return;
    setError(undefined);
    try {
      await createIntake.mutateAsync({
        source: "SOURCE_SENTRY_EXCEPTIONS",
        integrationId,
        resourceId: projectId,
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, SENTRY_CREATE_ERROR));
      return false;
    }
  };

  return {
    step,
    setStep,
    integrationId,
    setIntegrationId,
    projectId,
    setProjectId,
    skipInitialImport,
    setSkipInitialImport,
    connectOpen,
    setConnectOpen,
    connecting,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    projectsQuery,
    sentryIntegrations,
    sentryDefinition,
    existingNames,
    completeConnection,
    returnToConnection,
    connectSentry,
    createBoundIntake,
  };
}

const SENTRY_CONNECT_ERROR = "SuperPlane could not open the Sentry install page.";
const SENTRY_CREATE_ERROR = "SuperPlane could not create the Sentry intake.";

type SentryConnectParams = {
  organizationId: string;
  integrations: Array<{ metadata?: { id?: string } }>;
  integrationId: string;
  existingNames: Set<string>;
  createIntegration: ReturnType<typeof useCreateIntegration>;
  completeConnection: (integrationId: string) => void;
  setIntegrationId: (integrationId: string) => void;
  setConnectOpen: (open: boolean) => void;
  setError: (message?: string) => void;
};

// useSentryConnect opens a new Sentry connection. SuperPlane Cloud sends the
// browser to the public Sentry app, and a private app collects a token in the
// connect dialog.
function useSentryConnect(params: SentryConnectParams) {
  const location = useLocation();
  const [connecting, setConnecting] = useState(false);
  const returnPath = `${location.pathname}${location.search}`;

  const createConnection = async () => {
    const { result } = await createWithGeneratedName({
      baseName: "sentry",
      takenNames: params.existingNames,
      create: (name) =>
        params.createIntegration.mutateAsync({
          integrationName: "sentry",
          name,
          configuration: { setupReturnPath: returnPath },
        }),
    });
    return result.data?.integration;
  };

  const openInstallOrDialog = (integration: Awaited<ReturnType<typeof createConnection>>) => {
    if (integration?.status?.state === "ready" && integration.metadata?.id) {
      params.completeConnection(integration.metadata.id);
      return;
    }

    const action = integration?.status?.browserAction;
    rememberIntegrationSetupReturn(params.organizationId, returnPath);
    if (isHostedSentryInstallAction(action?.url)) {
      followBrowserAction(action);
      return;
    }

    params.setConnectOpen(true);
    if (integration?.metadata?.id) {
      params.setIntegrationId(integration.metadata.id);
    }
  };

  const connectSentry = async () => {
    params.setError(undefined);
    const readyId = readySentryConnectionId(params.integrations, params.integrationId);
    if (readyId) {
      params.completeConnection(readyId);
      return;
    }

    setConnecting(true);
    try {
      openInstallOrDialog(await createConnection());
    } catch (cause) {
      params.setError(getApiErrorMessage(cause, SENTRY_CONNECT_ERROR));
    } finally {
      setConnecting(false);
    }
  };

  return { connecting, connectSentry };
}

export function isHostedSentryInstallAction(url: string | undefined): boolean {
  if (!url) return false;
  return url.includes("/sentry/app/install") || url.includes("/sentry-apps/");
}

export function readySentryConnectionId(
  integrations: Array<{ metadata?: { id?: string } }>,
  selectedId: string,
): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
}

function useSentryConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const sentryIntegrations = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) =>
          integration.metadata?.integrationName === "sentry" &&
          integration.status?.state === "ready" &&
          integration.metadata.id,
      ),
    [connectedQuery.data],
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

  return {
    connectedQuery,
    sentryIntegrations,
    sentryDefinition: availableQuery.data?.find((integration) => integration.name === "sentry"),
    existingNames,
  };
}

export type SentryIntakeSetupModel = ReturnType<typeof useSentryIntakeSetup>;
