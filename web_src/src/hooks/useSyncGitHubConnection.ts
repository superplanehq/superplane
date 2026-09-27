import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { OrganizationsIntegration } from "@/api-client";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";
import { integrationKeys } from "@/hooks/useIntegrations";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

function replaceIntegration(
  integrations: OrganizationsIntegration[] | undefined,
  refreshed: OrganizationsIntegration,
): OrganizationsIntegration[] {
  const current = integrations ?? [];
  const integrationId = refreshed.metadata?.id;
  if (!integrationId) return current;

  const index = current.findIndex((integration) => integration.metadata?.id === integrationId);
  if (index === -1) return [...current, refreshed];

  return current.map((integration, currentIndex) => (currentIndex === index ? refreshed : integration));
}

/** Synchronizes one hosted GitHub connection and publishes the result as one cache update. */
export function useSyncGitHubConnection(organizationId: string) {
  const queryClient = useQueryClient();

  return useCallback(
    async (integrationId: string): Promise<OrganizationsIntegration> => {
      const response = await organizationsUpdateIntegration(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId, integrationId },
          body: {},
        }),
      );
      const refreshed = response.data?.integration;
      if (!refreshed) {
        throw new Error("GitHub connection synchronization returned no integration");
      }

      queryClient.setQueryData<OrganizationsIntegration[]>(integrationKeys.connected(organizationId), (current) =>
        replaceIntegration(current, refreshed),
      );
      return refreshed;
    },
    [organizationId, queryClient],
  );
}
