import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { usesHostedGitHubAppInstall } from "@/lib/integrations";

/**
 * Whether this installation can connect GitHub through the SuperPlane GitHub
 * App. Installations without the SUPERPLANE_GITHUB_APP_* environment variables
 * (for example, local development without a tunnel) cannot complete workspace
 * setup, so setup shows a blocking notice instead of the wizard.
 */
export function useGithubAppAvailability(organizationId: string): { resolved: boolean; available: boolean } {
  const definitions = useAvailableIntegrations({ enabled: !!organizationId, organizationId });
  const githubDefinition = (definitions.data ?? []).find((definition) => definition.name === "github");
  return {
    // Only a settled catalog can block setup; a transient fetch error must not.
    resolved: definitions.isSuccess,
    available: usesHostedGitHubAppInstall(githubDefinition),
  };
}
