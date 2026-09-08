import type { IntegrationsIntegrationDefinition } from "@/api-client";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { usesHostedGitHubAppInstall } from "@/lib/integrations";

export type GithubAppAvailability = { resolved: boolean; available: boolean };

/**
 * Whether this installation can connect GitHub through the SuperPlane GitHub
 * App. Installations without the SUPERPLANE_GITHUB_APP_* environment variables
 * (for example, local development without a tunnel) cannot complete workspace
 * setup, so setup shows a blocking notice instead of the wizard.
 */
export function githubAppAvailabilityFromCatalog(args: {
  isSuccess: boolean;
  isError: boolean;
  githubDefinition?: IntegrationsIntegrationDefinition;
}): GithubAppAvailability {
  return {
    // Only a successful catalog with no hosted app can block setup.
    // A fetch error must not leave the new-workspace page waiting forever.
    resolved: args.isSuccess || args.isError,
    available: args.isError || usesHostedGitHubAppInstall(args.githubDefinition),
  };
}

export function useGithubAppAvailability(organizationId: string): GithubAppAvailability {
  const definitions = useAvailableIntegrations({ enabled: !!organizationId, organizationId });
  const githubDefinition = (definitions.data ?? []).find((definition) => definition.name === "github");
  return githubAppAvailabilityFromCatalog({
    isSuccess: definitions.isSuccess,
    isError: definitions.isError,
    githubDefinition,
  });
}
