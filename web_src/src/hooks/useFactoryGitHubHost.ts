import { githubHostFromIntegration } from "@/lib/githubHost";
import { useOptionalFactoriesLayout } from "@/pages/factories/layout/factoriesLayoutContext";

import { useIntegration } from "./useIntegrations";

/** Hostname stored on the workspace GitHub integration. Empty means github.com. */
export function useFactoryGitHubHost(): string {
  const layout = useOptionalFactoriesLayout();
  const organizationId = layout?.organizationId ?? "";
  const onboarding = layout?.factory?.onboarding;
  const integrationId = onboarding?.vcsProvider === "bitbucket" ? "" : (onboarding?.vcsIntegrationId?.trim() ?? "");
  const integration = useIntegration(organizationId, integrationId);
  return githubHostFromIntegration(integration.data);
}
