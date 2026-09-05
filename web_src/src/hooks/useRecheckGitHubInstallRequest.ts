import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { OrganizationsIntegration } from "@/api-client";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";
import { integrationKeys } from "@/hooks/useIntegrations";
import { hostedGitHubInstallRequested } from "@/lib/hostedGitHubInstall";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

const RECHECK_INTERVAL_MS = 30_000;

export function pendingGitHubInstallRequestId(instances: OrganizationsIntegration[]): string | undefined {
  const pending = instances.find(
    (instance) =>
      instance.metadata?.integrationName === "github" &&
      instance.status?.state !== "ready" &&
      hostedGitHubInstallRequested(instance.status?.metadata),
  );
  return pending?.metadata?.id;
}

/**
 * Rechecks a pending GitHub App install request while the user waits.
 *
 * The GitHub approve callback carries no CSRF state and the installation
 * webhook cannot find a connection without an installation id, so the server
 * only learns about an approval during a connection sync. An empty update
 * runs that sync: it binds the approved installation and turns the
 * connection ready. Runs on page access and then every 30 seconds until the
 * request resolves or the page closes.
 */
export function useRecheckGitHubInstallRequest(organizationId: string, instances: OrganizationsIntegration[]) {
  const queryClient = useQueryClient();
  const integrationId = pendingGitHubInstallRequestId(instances);

  useEffect(() => {
    if (!organizationId || !integrationId) return;

    let cancelled = false;
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
    };

    void recheck();
    const interval = setInterval(() => void recheck(), RECHECK_INTERVAL_MS);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [organizationId, integrationId, queryClient]);
}
