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
  const location = useLocation();
  const [step, setStep] = useState<SentrySetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, sentryIntegrations, sentryDefinition, existingNames } = useSentryConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: Boolean(integrationId),
  });
  const issuesQuery = useIntegrationResources(
    organizationId,
    integrationId,
    "unresolved-issue",
    projectId ? { project: projectId } : undefined,
    { enabled: Boolean(integrationId && projectId) },
  );

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

  const createNewSentryConnection = async () => {
    const { result } = await createWithGeneratedName({
      baseName: "sentry",
      takenNames: existingNames,
      create: (name) =>
        createIntegration.mutateAsync({
          integrationName: "sentry",
          name,
          configuration: { setupReturnPath: `${location.pathname}${location.search}` },
        }),
    });
    return result.data?.integration;
  };

  const openSentryInstallOrDialog = (integration: Awaited<ReturnType<typeof createNewSentryConnection>>) => {
    if (integration?.status?.state === "ready" && integration.metadata?.id) {
      completeConnection(integration.metadata.id);
      return;
    }

    const action = integration?.status?.browserAction;
    rememberIntegrationSetupReturn(organizationId, `${location.pathname}${location.search}`);
    if (isHostedSentryInstallAction(action?.url)) {
      followBrowserAction(action);
      return;
    }

    setConnectOpen(true);
    if (integration?.metadata?.id) {
      setIntegrationId(integration.metadata.id);
    }
  };

  const connectSentry = async () => {
    setError(undefined);
    const readyId = readySentryConnectionId(sentryIntegrations, integrationId);
    if (readyId) {
      completeConnection(readyId);
      return;
    }

    setConnecting(true);
    try {
      openSentryInstallOrDialog(await createNewSentryConnection());
    } catch (cause) {
      setError(getApiErrorMessage(cause, SENTRY_CONNECT_ERROR));
    } finally {
      setConnecting(false);
    }
  };

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
    connectOpen,
    setConnectOpen,
    connecting,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    projectsQuery,
    issuesQuery,
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
