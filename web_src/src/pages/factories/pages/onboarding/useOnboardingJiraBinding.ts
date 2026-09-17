import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useEffect, useState } from "react";

import { readOnboardingJiraProject, writeOnboardingJiraProject } from "./onboardingJiraProject";

export function useOnboardingJiraBinding(
  organizationId: string,
  factoryId: string,
  jiraSelection?: { id: string; ready: boolean },
) {
  const jiraIntegrationId = jiraSelection?.ready ? jiraSelection.id : "";
  const [jiraProjectId, setJiraProjectIdState] = useState(() =>
    readOnboardingJiraProject(factoryId, jiraIntegrationId),
  );
  const jiraProjectsQuery = useIntegrationResources(organizationId, jiraIntegrationId, "project", undefined, {
    enabled: Boolean(jiraIntegrationId),
  });

  useEffect(() => {
    setJiraProjectIdState(readOnboardingJiraProject(factoryId, jiraIntegrationId));
  }, [factoryId, jiraIntegrationId]);

  const setJiraProjectId = useCallback(
    (projectId: string) => {
      setJiraProjectIdState(projectId);
      writeOnboardingJiraProject(factoryId, jiraIntegrationId, projectId);
    },
    [factoryId, jiraIntegrationId],
  );

  return {
    jiraIntegrationId,
    jiraProjectId,
    setJiraProjectId,
    jiraProjects: jiraProjectsQuery.data ?? [],
    jiraProjectsLoading: jiraProjectsQuery.isPending,
    jiraProjectsError: jiraProjectsQuery.isError,
    retryJiraProjects: () => {
      void jiraProjectsQuery.refetch();
    },
  };
}
