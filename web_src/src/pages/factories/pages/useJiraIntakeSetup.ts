import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useRef, useState } from "react";

export type JiraSetupStep = "connection" | "project" | "complete";

export function useJiraIntakeSetup(organizationId: string, factoryId: string, open: boolean, selectIntegrationId = "") {
  const [step, setStep] = useState<JiraSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [error, setError] = useState<string>();
  const pickedReturnedConnection = useRef(false);

  const { connectedQuery, jiraIntegrations, jiraConnections, jiraDefinition, existingNames } =
    useJiraConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project");

  useEffect(() => {
    if (!open) {
      pickedReturnedConnection.current = false;
      return;
    }
    setStep("connection");
    setIntegrationId("");
    setProjectId("");
    setError(undefined);
  }, [open]);

  useEffect(() => {
    if (!open || !selectIntegrationId || pickedReturnedConnection.current) return;

    const returned = jiraConnections.find((integration) => integration.metadata?.id === selectIntegrationId);
    if (!returned || returned.status?.state !== "ready") return;

    pickedReturnedConnection.current = true;
    setIntegrationId(selectIntegrationId);
    setStep("project");
  }, [open, selectIntegrationId, jiraConnections]);

  useEffect(() => {
    if (selectIntegrationId) return;
    if (!integrationId && jiraIntegrations.length === 1) {
      setIntegrationId(jiraIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, jiraIntegrations, selectIntegrationId]);

  const completeConnection = (connectedIntegrationId: string) => {
    setIntegrationId(connectedIntegrationId);
    setConnectOpen(false);
    setStep("project");
    void connectedQuery.refetch();
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
      setStep("complete");
    } catch (cause) {
      setError(getApiErrorMessage(cause, "SuperPlane could not create the Jira intake."));
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
    createBoundIntake,
  };
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
