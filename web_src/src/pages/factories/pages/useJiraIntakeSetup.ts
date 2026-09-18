import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useRef, useState } from "react";

import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";

export type JiraSetupStep = "connection" | "project";

export function useJiraIntakeSetup(organizationId: string, factoryId: string, selectIntegrationId = "") {
  const [step, setStep] = useState<JiraSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [stayOnConnection, setStayOnConnection] = useState(false);
  const [error, setError] = useState<string>();
  const pickedReturnedConnection = useRef(false);

  const { connectedQuery, jiraIntegrations, jiraConnections, jiraDefinition, existingNames } =
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
    if (stayOnConnection || step !== "connection") {
      return;
    }
    if (selectIntegrationId) {
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

  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
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
        source: "SOURCE_JIRA_ISSUES",
        integrationId,
        resourceId: projectId,
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
    setProjectId,
    connectOpen,
    setConnectOpen,
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
    createBoundIntake,
  };
}

export function readyJiraConnectionId(integrations: Array<{ metadata?: { id?: string } }>, selectedId: string): string {
  if (selectedId && integrations.some((integration) => integration.metadata?.id === selectedId)) {
    return selectedId;
  }
  return integrations[0]?.metadata?.id ?? "";
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
    jiraIntegrations,
    jiraConnections,
    jiraDefinition: availableQuery.data?.find((integration) => integration.name === "jira"),
    existingNames,
  };
}

export type JiraIntakeSetupModel = ReturnType<typeof useJiraIntakeSetup>;
