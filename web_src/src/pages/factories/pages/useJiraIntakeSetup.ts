import { useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useMemo, useState } from "react";

export type JiraSetupStep = "connection" | "project" | "complete";

export function useJiraIntakeSetup(organizationId: string, factoryId: string, open: boolean) {
  const [step, setStep] = useState<JiraSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [error, setError] = useState<string>();

  const { connectedQuery, jiraIntegrations, jiraDefinition, existingNames } = useJiraConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project");

  useEffect(() => {
    if (open) {
      setStep("connection");
      setIntegrationId("");
      setProjectId("");
      setError(undefined);
    }
  }, [open]);

  useEffect(() => {
    if (!integrationId && jiraIntegrations.length === 1) {
      setIntegrationId(jiraIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, jiraIntegrations]);

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

  const jiraIntegrations = useMemo(
    () =>
      (connectedQuery.data ?? []).filter(
        (integration) =>
          integration.metadata?.integrationName === "jira" &&
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
    jiraIntegrations,
    jiraDefinition: availableQuery.data?.find((integration) => integration.name === "jira"),
    existingNames,
  };
}

export type JiraIntakeSetupModel = ReturnType<typeof useJiraIntakeSetup>;
