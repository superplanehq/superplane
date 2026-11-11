import type { FactoriesFactoryAgentResource, FactoriesFactoryOnboarding } from "@/api-client";
import { useQueries } from "@tanstack/react-query";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { integrationResourcesQueryOptions } from "@/hooks/useIntegrations";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";

/** Keep repository catalogs separate so other integration pickers share the cache. */
export function useReviewRequestReviewers(organizationId: string, factoryId: string, integrationId: string) {
  const layout = useOptionalFactoriesLayout();
  const contextRepos = useFactoryAgentResources(organizationId, factoryId, "KIND_CONTEXT_REPO");
  const repositories = workspaceRepositoryNames(layout?.factory?.onboarding, contextRepos.data ?? []);
  const queries = useQueries({
    queries: repositories.map((repository) =>
      integrationResourcesQueryOptions(organizationId, integrationId, "human_reviewer", { repository }),
    ),
  });
  return {
    reviewers: reviewerLogins(queries.flatMap((query) => query.data ?? [])),
    isLoading: contextRepos.isLoading || queries.some((query) => query.isLoading),
    isError:
      contextRepos.isError || queries.some((query) => query.isError) || !integrationId || repositories.length === 0,
    refetch: () => {
      void contextRepos.refetch();
      for (const query of queries) void query.refetch();
    },
  };
}

function workspaceRepositoryNames(
  onboarding: FactoriesFactoryOnboarding | undefined,
  resources: FactoriesFactoryAgentResource[],
) {
  return [
    ...new Set(
      [
        onboarding?.appRepository,
        onboarding?.backlogRepository,
        ...resources.filter((repo) => repo.enabled !== false).map((repo) => repo.repository),
      ].flatMap((repository) => (repository?.trim() ? [repository.trim()] : [])),
    ),
  ];
}

function reviewerLogins(resources: Array<{ id?: string; name?: string }>): string[] {
  const reviewers = new Map<string, string>();
  for (const resource of resources) {
    const login = resource.id?.trim() || resource.name?.trim();
    if (login) reviewers.set(login.toLowerCase(), login);
  }
  return [...reviewers.values()].sort((left, right) => left.localeCompare(right));
}
