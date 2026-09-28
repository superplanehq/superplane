import { useCallback, useEffect, useState } from "react";
import type { OrganizationsIntegration } from "@/api-client";
import { pendingGitHubRequestConnection } from "@/lib/startDirectGitHubConnect";

import { useSyncGitHubConnection } from "./useSyncGitHubConnection";

export const INSTALL_REQUEST_RECHECK_INTERVAL_MS = 5_000;
export const INSTALLATION_DISCOVERY_RECHECK_INTERVAL_MS = 1_000;

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
export function useRecheckGitHubInstallRequest(
  organizationId: string,
  integrationId?: string,
  enabled = true,
  intervalMs = INSTALL_REQUEST_RECHECK_INTERVAL_MS,
) {
  const syncGitHubConnection = useSyncGitHubConnection(organizationId);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const retry = useCallback(() => {
    setFailed(false);
    setAttempt((value) => value + 1);
  }, []);

  useEffect(() => {
    if (!organizationId || !integrationId || !enabled) {
      setFailed(false);
      return;
    }

    let cancelled = false;
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const recheck = async () => {
      try {
        await syncGitHubConnection(integrationId);
      } catch {
        if (!cancelled) setFailed(true);
        return;
      }
      if (cancelled) return;
      setFailed(false);
      timeout = setTimeout(() => void recheck(), intervalMs);
    };

    void recheck();
    return () => {
      cancelled = true;
      if (timeout) clearTimeout(timeout);
    };
  }, [attempt, enabled, integrationId, intervalMs, organizationId, syncGitHubConnection]);

  return { failed, retry };
}
