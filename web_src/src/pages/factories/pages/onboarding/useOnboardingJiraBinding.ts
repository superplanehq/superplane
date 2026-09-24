import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useEffect, useMemo, useState } from "react";

import type { JiraCompletionColumnValue } from "../jiraCompletionColumn";
import { preferredJiraCompletionColumn } from "../jiraCompletionColumn";
import { DEFAULT_JIRA_COMPLETION_SETTINGS } from "../intakeSourceSettingsModel";
import { readOnboardingJiraProject, writeOnboardingJiraProject } from "./onboardingJiraProject";

function jiraStatusColumnNames(statuses: Array<{ name?: string; id?: string }> | undefined): string[] {
  return (statuses ?? [])
    .map((resource) => resource.name?.trim() || resource.id?.trim() || "")
    .filter((name) => name.length > 0);
}

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
  const jiraStatusesQuery = useIntegrationResources(
    organizationId,
    jiraIntegrationId,
    "issueStatus",
    jiraProjectId ? { project: jiraProjectId } : undefined,
    { enabled: Boolean(jiraIntegrationId && jiraProjectId) },
  );
  const jiraStatusColumns = useMemo(() => jiraStatusColumnNames(jiraStatusesQuery.data), [jiraStatusesQuery.data]);
  const jiraCompletionAutoResolved = useMemo(() => {
    if (!jiraProjectId || jiraStatusesQuery.isPending || jiraStatusesQuery.isError) {
      return false;
    }
    if (jiraStatusColumns.length === 0) {
      return false;
    }
    return preferredJiraCompletionColumn(jiraStatusColumns, "") !== "";
  }, [jiraProjectId, jiraStatusColumns, jiraStatusesQuery.isError, jiraStatusesQuery.isPending]);
  const jiraCompletionNeedsManualColumn = Boolean(jiraProjectId && !jiraCompletionAutoResolved);

  useEffect(() => {
    setJiraProjectIdState(readOnboardingJiraProject(factoryId, jiraIntegrationId));
    setJiraCompletionState({ ...DEFAULT_JIRA_COMPLETION_SETTINGS });
  }, [factoryId, jiraIntegrationId]);

  useEffect(() => {
    if (!jiraProjectId || !jiraCompletion.jiraMoveOnComplete) {
      return;
    }
    const preferred = preferredJiraCompletionColumn(jiraStatusColumns, jiraCompletion.jiraCompletionColumn);
    if (preferred && preferred !== jiraCompletion.jiraCompletionColumn) {
      setJiraCompletionState((current) => ({ ...current, jiraCompletionColumn: preferred }));
    }
  }, [jiraCompletion.jiraCompletionColumn, jiraCompletion.jiraMoveOnComplete, jiraProjectId, jiraStatusColumns]);

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
    jiraCompletionNeedsManualColumn,
    jiraProjects: jiraProjectsQuery.data ?? [],
    jiraProjectsLoading: jiraProjectsQuery.isPending,
    jiraProjectsError: jiraProjectsQuery.isError,
    retryJiraProjects: () => {
      void jiraProjectsQuery.refetch();
    },
  };
}
