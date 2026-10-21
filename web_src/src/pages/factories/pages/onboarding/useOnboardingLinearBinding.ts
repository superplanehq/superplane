import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useCallback, useEffect, useState } from "react";

import { EMPTY_LINEAR_INTAKE_SCOPE, type LinearIntakeScope } from "../linearIntakeScope";
import { readOnboardingLinearScope, writeOnboardingLinearScope } from "./onboardingLinearScope";

export function useOnboardingLinearBinding(
  organizationId: string,
  factoryId: string,
  linearSelection?: { id: string; ready: boolean },
) {
  const linearIntegrationId = linearSelection?.ready ? linearSelection.id : "";
  const [linearScope, setLinearScope] = useState(() => readOnboardingLinearScope(factoryId, linearIntegrationId));
  const teamsQuery = useIntegrationResources(organizationId, linearIntegrationId, "team", undefined, {
    enabled: Boolean(linearIntegrationId),
  });
  const projectsQuery = useIntegrationResources(
    organizationId,
    linearIntegrationId,
    "project",
    { team: linearScope.teamId },
    { enabled: Boolean(linearIntegrationId && linearScope.teamId) },
  );

  useEffect(() => {
    setLinearScope(readOnboardingLinearScope(factoryId, linearIntegrationId));
  }, [factoryId, linearIntegrationId]);

  const saveScope = useCallback(
    (next: LinearIntakeScope) => {
      writeOnboardingLinearScope(factoryId, linearIntegrationId, next);
      setLinearScope(next);
    },
    [factoryId, linearIntegrationId],
  );

  // Drop a saved team or project that Linear no longer returns. Keep it while
  // the list loads or fails, so a slow answer does not erase the choice.
  const teamsReady = Boolean(linearIntegrationId) && !teamsQuery.isPending && !teamsQuery.isError;
  useEffect(() => {
    if (!teamsReady || !linearScope.teamId) return;
    const available = (teamsQuery.data ?? []).some((team) => team.id === linearScope.teamId);
    if (!available) saveScope(EMPTY_LINEAR_INTAKE_SCOPE);
  }, [teamsReady, teamsQuery.data, linearScope.teamId, saveScope]);

  const projectsReady = Boolean(linearScope.teamId) && !projectsQuery.isPending && !projectsQuery.isError;
  useEffect(() => {
    if (!projectsReady) return;
    const available = new Set(
      (projectsQuery.data ?? []).map((project) => project.id).filter((id): id is string => Boolean(id)),
    );
    const projectIds = linearScope.projectIds.filter((id) => available.has(id));
    if (projectIds.length !== linearScope.projectIds.length) saveScope({ ...linearScope, projectIds });
  }, [projectsReady, projectsQuery.data, linearScope, saveScope]);

  return {
    linearIntegrationId,
    linearScope,
    changeLinearScope: saveScope,
  };
}
