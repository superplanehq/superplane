import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { usesHostedJiraOAuth } from "@/lib/integrations";
import { startDirectJiraConnect } from "@/lib/startDirectJiraConnect";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "react-router";

import type { JiraCompletionColumnValue } from "./jiraCompletionColumn";
import { DEFAULT_JIRA_COMPLETION_SETTINGS, jiraCompletionSettingsToApi } from "./intakeSourceSettingsModel";
import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";

export type JiraSetupStep = "connection" | "project";

export function useJiraIntakeSetup(organizationId: string, factoryId: string, selectIntegrationId = "") {
  const [step, setStep] = useState<JiraSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [skipInitialImport, setSkipInitialImport] = useState(false);
  const [jiraCompletion, setJiraCompletion] = useState<JiraCompletionColumnValue>({
    ...DEFAULT_JIRA_COMPLETION_SETTINGS,
  });
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();
  const pickedReturnedConnection = useRef(false);

  const { connectedQuery, availableQuery, jiraIntegrations, jiraConnections, jiraDefinition, existingNames } =
    useJiraConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project", undefined, {
    enabled: Boolean(integrationId),
  });

  useEffect(() => {
    pickedReturnedConnection.current = false;
  }, [selectIntegrationId]);

  useEffect(() => {
    if (!selectIntegrationId || pickedReturnedConnection.current) {
      return;
    }

    const returned = jiraConnections.find((integration) => integration.metadata?.id === selectIntegrationId);
    if (!returned || returned.status?.state !== "ready") {
      return;
    }

    pickedReturnedConnection.current = true;
    setIntegrationId(selectIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [selectIntegrationId, jiraConnections, connectedQuery]);

  useEffect(() => {
    if (stayOnConnection || step !== "connection" || selectIntegrationId) {
      return;
    }
    const readyId = readyJiraConnectionId(jiraIntegrations, integrationId);
    if (!readyId) {
      return;
    }
    setIntegrationId(readyId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  }, [stayOnConnection, step, selectIntegrationId, jiraIntegrations, integrationId, connectedQuery]);

  const selectProject = (id: string) => {
    setProjectId(id);
    setJiraCompletion({ ...DEFAULT_JIRA_COMPLETION_SETTINGS });
  };
  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
  };

  const { connecting, connectJira } = useJiraConnect({
    organizationId,
    integrations: jiraIntegrations,
    integrationId,
    existingNames,
    jiraDefinition,
    definitionLoading: availableQuery.isLoading,
    createIntegration,
    completeConnection,
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
        source: "SOURCE_JIRA_ISSUES",
        integrationId,
        resourceId: projectId,
        settings: jiraCompletionSettingsToApi(jiraCompletion),
        ...(skipInitialImport ? { skipInitialImport: true } : {}),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, JIRA_INTAKE_SETUP_COPY.wizardCreateError));
      return false;
    }
  };

  return {
    step,
    setStep,
    integrationId,
    setIntegrationId,
    projectId,
    setProjectId: selectProject,
    skipInitialImport,
    setSkipInitialImport,
    jiraCompletion,
    setJiraCompletion,
    connectOpen,
    setConnectOpen,
    connecting,
    error,
    connectedQuery,
    createIntegration,
    createIntake,
    projectsQuery,
    jiraIntegrations,
    jiraDefinition,
    existingNames,
    completeConnection,
    returnToConnection,
    connectJira,
    createBoundIntake,
  };
}

export function readyJiraConnectionId(integrations: Array<{ metadata?: { id?: string } }>, selectedId: string): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
}

type JiraConnectParams = {
  organizationId: string;
  integrations: Array<{ metadata?: { id?: string } }>;
  integrationId: string;
  existingNames: Set<string>;
  jiraDefinition?: IntegrationsIntegrationDefinition;
  definitionLoading: boolean;
  createIntegration: ReturnType<typeof useCreateIntegration>;
  completeConnection: (integrationId: string) => void;
  setConnectOpen: (open: boolean) => void;
  setError: (message?: string) => void;
};

// Hosted Jira OAuth opens Atlassian in this tab. A private Atlassian app still
// collects Client ID and Client Secret in the connect dialog.
function useJiraConnect(params: JiraConnectParams) {
  const location = useLocation();
  const [connecting, setConnecting] = useState(false);
  const returnPath = `${location.pathname}${location.search}`;

  const connectJira = async () => {
    params.setError(undefined);
    const readyId = readyJiraConnectionId(params.integrations, params.integrationId);
    if (readyId) {
      params.completeConnection(readyId);
      return;
    }
    if (params.definitionLoading) {
      return;
    }
    if (!usesHostedJiraOAuth(params.jiraDefinition)) {
      params.setConnectOpen(true);
      return;
    }

    setConnecting(true);
    try {
      await startDirectJiraConnect({
        organizationId: params.organizationId,
        returnTo: returnPath,
        existingNames: params.existingNames,
        create: async (payload) => {
          const response = await params.createIntegration.mutateAsync(payload);
          return response.data;
        },
      });
    } catch (cause) {
      params.setError(getApiErrorMessage(cause, JIRA_INTAKE_SETUP_COPY.wizardConnectError));
    } finally {
      setConnecting(false);
    }
  };

  return { connecting, connectJira };
}

function useJiraConnections(organizationId: string) {
  const connectedQuery = useConnectedIntegrations(organizationId);
  const availableQuery = useAvailableIntegrations({ organizationId });

  const jiraConnections = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) => integration.metadata?.integrationName === "jira" && integration.metadata.id,
      ),
    [connectedQuery.data],
  );
  const jiraIntegrations = useMemo(
    () => jiraConnections.filter((integration) => integration.status?.state === "ready"),
    [jiraConnections],
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
    availableQuery,
    jiraIntegrations,
    jiraConnections,
    jiraDefinition: availableQuery.data?.find((integration) => integration.name === "jira"),
    existingNames,
  };
}

export type JiraIntakeSetupModel = ReturnType<typeof useJiraIntakeSetup>;
