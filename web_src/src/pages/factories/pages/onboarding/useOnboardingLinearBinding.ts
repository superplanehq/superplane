import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useEffect, useState } from "react";

import { readOnboardingLinearProjects, writeOnboardingLinearProjects } from "./onboardingLinearProjects";

export function useOnboardingLinearBinding(
  organizationId: string,
  factoryId: string,
  linearSelection?: { id: string; ready: boolean },
) {
  const linearIntegrationId = linearSelection?.ready ? linearSelection.id : "";
  const [linearProjectIds, setLinearProjectIds] = useState(() =>
    readOnboardingLinearProjects(factoryId, linearIntegrationId),
  );
  const linearProjectsQuery = useIntegrationResources(organizationId, linearIntegrationId, "project", undefined, {
    enabled: Boolean(linearIntegrationId),
  });

  useEffect(() => {
    setLinearProjectIds(readOnboardingLinearProjects(factoryId, linearIntegrationId));
  }, [factoryId, linearIntegrationId]);

  const toggleLinearProject = useCallback(
    (projectId: string) => {
      setLinearProjectIds((current) => {
        const next = current.includes(projectId) ? current.filter((id) => id !== projectId) : [...current, projectId];
        writeOnboardingLinearProjects(factoryId, linearIntegrationId, next);
        return next;
      });
    },
    [factoryId, linearIntegrationId],
  );

  return {
    linearIntegrationId,
    linearProjectIds,
    toggleLinearProject,
    linearProjects: linearProjectsQuery.data ?? [],
    linearProjectsLoading: linearProjectsQuery.isPending,
    linearProjectsError: linearProjectsQuery.isError,
    retryLinearProjects: () => {
      void linearProjectsQuery.refetch();
    },
  };
}
