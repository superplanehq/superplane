import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";

export type GithubAppAvailability = { resolved: boolean; available: boolean; failed: boolean };

/**
 * Whether this process holds the public SuperPlane GitHub App. Installations
 * without SUPERPLANE_GITHUB_APP_* (for example, local development without a
 * tunnel) cannot complete workspace setup, so setup shows a blocking notice
 * instead of the wizard. Per-organization hosted install is not this check.
 */
export function githubAppAvailabilityFromCatalog(args: {
  isSuccess: boolean;
  isError: boolean;
  githubAppConfigured?: boolean;
  githubDefinition?: IntegrationsIntegrationDefinition;
}): GithubAppAvailability {
  return {
    // A fetch error is settled so the page can show retry. It is not
    // available, so setup does not create a workspace it cannot finish.
    resolved: args.isSuccess || args.isError,
    failed: args.isError,
    available: !args.isError && args.githubAppConfigured === true,
  };
}

export function useGithubAppAvailability(organizationId: string): GithubAppAvailability & {
  retry: () => Promise<void>;
} {
  const catalog = useAvailableIntegrations({ enabled: !!organizationId, organizationId });
  return {
    ...githubAppAvailabilityFromCatalog({
      isSuccess: catalog.isSuccess,
      isError: catalog.isError,
      githubAppConfigured: catalog.githubAppConfigured,
    }),
    retry: async () => {
      await catalog.refetch();
    },
  };
}
