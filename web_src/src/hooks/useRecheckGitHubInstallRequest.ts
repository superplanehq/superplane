import { useEffect } from "react";
import type { OrganizationsIntegration } from "@/api-client";
import { pendingGitHubRequestConnection } from "@/lib/startDirectGitHubConnect";

import { useSyncGitHubConnection } from "./useSyncGitHubConnection";

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
  const syncGitHubConnection = useSyncGitHubConnection(organizationId);

  useEffect(() => {
    if (!organizationId || !integrationId || !enabled) return;

    let cancelled = false;
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const recheck = async () => {
      try {
        await syncGitHubConnection(integrationId);
      } catch {
        // The connection stays in the waiting state; the next tick retries.
      }
      if (cancelled) return;
      if (!cancelled) timeout = setTimeout(() => void recheck(), RECHECK_INTERVAL_MS);
    };

    void recheck();
    return () => {
      cancelled = true;
      if (timeout) clearTimeout(timeout);
    };
  }, [enabled, integrationId, organizationId, syncGitHubConnection]);
}
