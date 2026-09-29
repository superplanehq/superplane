import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { OrganizationsIntegration } from "@/api-client";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";
import { integrationKeys } from "@/hooks/useIntegrations";
import { pendingGitHubRequestConnection } from "@/lib/startDirectGitHubConnect";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

const RECHECK_INTERVAL_MS = 5_000;

export function pendingGitHubInstallRequestId(
  instances: OrganizationsIntegration[],
  currentUserId?: string,
  preferredIntegrationId?: string,
): string | undefined {
  return pendingGitHubRequestConnection(instances, currentUserId, preferredIntegrationId)?.id;
}

/**
 * Rechecks a pending GitHub App install request while the user waits.
 *
 * The GitHub approve callback carries no CSRF state and the installation
 * webhook cannot find a connection without an installation id, so the server
 * only learns about an approval during a connection sync. An empty update
 * runs that sync: an approved installation joins the account picker, where
 * the user picks it. Runs on page access and then every five seconds until the
 * request resolves or the page closes.
 */
export function useRecheckGitHubInstallRequest(organizationId: string, integrationId?: string, enabled = true) {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!organizationId || !integrationId || !enabled) return;

    let cancelled = false;
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const recheck = async () => {
      try {
        await organizationsUpdateIntegration(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId, integrationId },
            body: {},
          }),
        );
      } catch {
        // The connection stays in the waiting state; the next tick retries.
      }
      if (cancelled) return;
      await queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
      if (!cancelled) timeout = setTimeout(() => void recheck(), RECHECK_INTERVAL_MS);
    };

    void recheck();
    return () => {
      cancelled = true;
      if (timeout) clearTimeout(timeout);
    };
  }, [enabled, organizationId, integrationId, queryClient]);
}
