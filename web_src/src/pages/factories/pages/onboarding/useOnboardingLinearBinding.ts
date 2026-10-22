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
  const [binding, setBinding] = useState(() => ({
    factoryId,
    integrationId: linearIntegrationId,
    scope: readOnboardingLinearScope(factoryId, linearIntegrationId),
  }));

  // Rekey the scope during render when the connection changes. An effect
  // would run the checks below once with the scope of the previous connection
  // and could erase the stored scope of the new one.
  if (binding.factoryId !== factoryId || binding.integrationId !== linearIntegrationId) {
    setBinding({
      factoryId,
      integrationId: linearIntegrationId,
      scope: readOnboardingLinearScope(factoryId, linearIntegrationId),
    });
  }
  const linearScope = binding.scope;

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

  const saveScope = useCallback(
    (next: LinearIntakeScope) => {
      writeOnboardingLinearScope(factoryId, linearIntegrationId, next);
      setBinding({ factoryId, integrationId: linearIntegrationId, scope: next });
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
    // An empty project list accepts every ticket in the team. When every saved
    // project is gone, keep the selection instead of widening the intake
    // silently. The scope fields show the missing projects, so the user picks
    // a replacement or clears them.
    if (projectIds.length === 0 || projectIds.length === linearScope.projectIds.length) return;
    saveScope({ ...linearScope, projectIds });
  }, [projectsReady, projectsQuery.data, linearScope, saveScope]);

  return {
    linearIntegrationId,
    linearScope,
    changeLinearScope: saveScope,
  };
}
