import { useQuery } from "@tanstack/react-query";

import { integrationKeys, resolveRepositoryDefaultBranch } from "@/hooks/useIntegrations";

type CatalogRepository = {
  fullName?: string | null;
  defaultBranch?: string | null;
};

/**
 * The default branch of a repository on the host. The catalog value shows
 * first; the live lookup replaces it when it returns.
 */
export function useRepositoryDefaultBranch(args: {
  organizationId: string;
  integrationId: string;
  repository: string;
  catalogRepositories?: CatalogRepository[];
}): string {
  const live = useQuery({
    queryKey: [
      ...integrationKeys.resources(args.organizationId, args.integrationId, "default_branch", {
        repository: args.repository,
      }),
      "name",
    ],
    queryFn: () => resolveRepositoryDefaultBranch(args.organizationId, args.integrationId, args.repository),
    enabled: Boolean(args.organizationId && args.integrationId && args.repository),
    staleTime: 2 * 60 * 1000,
    retry: false,
  });
  const catalog =
    args.catalogRepositories
      ?.find((candidate) => candidate.fullName?.toLowerCase() === args.repository.toLowerCase())
      ?.defaultBranch?.trim() ?? "";
  return live.data?.trim() || catalog;
}
