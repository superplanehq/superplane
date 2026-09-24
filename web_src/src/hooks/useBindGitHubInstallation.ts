import { useMutation, useQueryClient } from "@tanstack/react-query";

import { integrationKeys } from "@/hooks/useIntegrations";
import { bindHostedGitHubInstallation } from "@/lib/hostedGitHubInstall";

/**
 * Binds a pending GitHub connection to a chosen App installation in place.
 * A full-page redirect through the bind endpoint reloads the whole app, so
 * this mutation calls the endpoint from the page and refreshes the
 * connection list before it resolves.
 */
export function useBindGitHubInstallation(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ state, installationId }: { state: string; installationId: string }) => {
      await bindHostedGitHubInstallation(state, installationId);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
    },
  });
}
