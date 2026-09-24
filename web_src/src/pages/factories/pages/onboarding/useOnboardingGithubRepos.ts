import { useMemo } from "react";

import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";

export function useOnboardingGithubRepos(organizationId: string, githubIntegrationId: string) {
  const githubIntegration = useIntegration(organizationId, githubIntegrationId);
  const resources = useIntegrationResources(organizationId, githubIntegrationId, "repository");
  const repositories = useMemo(
    () =>
      (resources.data ?? [])
        .map((resource) => resource.name ?? resource.id ?? "")
        .filter((repository): repository is string => Boolean(repository)),
    [resources.data],
  );

  return {
    githubIntegration,
    repositories,
    // A background refresh keeps the last successful list visible. Only the
    // first request replaces the picker with the full loading state.
    repositoriesLoading: resources.isPending,
    repositoriesError: resources.error,
  };
}
