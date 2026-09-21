import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useEffect, useState } from "react";

import type { JiraCompletionColumnValue } from "../jiraCompletionColumn";
import { DEFAULT_JIRA_COMPLETION_SETTINGS } from "../intakeSourceSettingsModel";
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
  const [jiraCompletion, setJiraCompletionState] = useState<JiraCompletionColumnValue>({
    ...DEFAULT_JIRA_COMPLETION_SETTINGS,
  });
  const jiraProjectsQuery = useIntegrationResources(organizationId, jiraIntegrationId, "project", undefined, {
    enabled: Boolean(jiraIntegrationId),
  });

  useEffect(() => {
    setJiraProjectIdState(readOnboardingJiraProject(factoryId, jiraIntegrationId));
    setJiraCompletionState({ ...DEFAULT_JIRA_COMPLETION_SETTINGS });
  }, [factoryId, jiraIntegrationId]);

  const setJiraProjectId = useCallback(
    (projectId: string) => {
      setJiraProjectIdState(projectId);
      writeOnboardingJiraProject(factoryId, jiraIntegrationId, projectId);
      setJiraCompletionState({ ...DEFAULT_JIRA_COMPLETION_SETTINGS });
    },
    [factoryId, jiraIntegrationId],
  );

  const setJiraCompletion = useCallback((next: JiraCompletionColumnValue) => {
    setJiraCompletionState(next);
  }, []);

  return {
    jiraIntegrationId,
    jiraProjectId,
    setJiraProjectId,
    jiraCompletion,
    setJiraCompletion,
    jiraProjects: jiraProjectsQuery.data ?? [],
    jiraProjectsLoading: jiraProjectsQuery.isPending,
    jiraProjectsError: jiraProjectsQuery.isError,
    retryJiraProjects: () => {
      void jiraProjectsQuery.refetch();
    },
  };
}
