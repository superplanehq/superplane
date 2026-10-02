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

  const projectsReady = Boolean(linearIntegrationId) && !linearProjectsQuery.isPending && !linearProjectsQuery.isError;
  useEffect(() => {
    if (!projectsReady) return;
    const available = new Set(
      (linearProjectsQuery.data ?? []).map((project) => project.id).filter((id): id is string => Boolean(id)),
    );
    setLinearProjectIds((current) => {
      const next = current.filter((id) => available.has(id));
      if (next.length === current.length && next.every((id, index) => id === current[index])) return current;
      writeOnboardingLinearProjects(factoryId, linearIntegrationId, next);
      return next;
    });
  }, [projectsReady, linearProjectsQuery.data, factoryId, linearIntegrationId]);

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
