import type { OrganizationsIntegration } from "@/api-client";
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

export function useJiraIntakeSetup(organizationId: string, factoryId: string, open: boolean, selectNewest = false) {
  const [step, setStep] = useState<JiraSetupStep>("connection");
  const [integrationId, setIntegrationId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [connectOpen, setConnectOpen] = useState(false);
  const [error, setError] = useState<string>();
  const pickedNewest = useRef(false);

  const { connectedQuery, jiraIntegrations, jiraConnections, jiraDefinition, existingNames } =
    useJiraConnections(organizationId);
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const createIntake = useCreateFactoryIntake(organizationId, factoryId);
  const projectsQuery = useIntegrationResources(organizationId, integrationId, "project");

  useEffect(() => {
    if (!open) {
      pickedNewest.current = false;
      return;
    }
    setStep("connection");
    setIntegrationId("");
    setProjectId("");
    setError(undefined);
  }, [open]);

  useEffect(() => {
    if (!open || !selectNewest || pickedNewest.current) return;

    const newest = newestJiraConnection(jiraConnections);
    if (!newest || newest.status?.state !== "ready") return;
    const id = newest.metadata?.id;
    if (!id) return;

    pickedNewest.current = true;
    setIntegrationId(id);
    setStep("project");
  }, [open, selectNewest, jiraConnections]);

  useEffect(() => {
    if (selectNewest) return;
    if (!integrationId && jiraIntegrations.length === 1) {
      setIntegrationId(jiraIntegrations[0].metadata?.id ?? "");
    }
  }, [integrationId, jiraIntegrations, selectNewest]);

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

function newestJiraConnection(integrations: OrganizationsIntegration[]): OrganizationsIntegration | undefined {
  return [...integrations].sort((left, right) => {
    const leftAt = left.metadata?.createdAt ?? "";
    const rightAt = right.metadata?.createdAt ?? "";
    return rightAt.localeCompare(leftAt);
  })[0];
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
