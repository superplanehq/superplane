import type { MeDescribeVcsProviderOnboardingResponse } from "@/api-client";
import { useEffect, useState } from "react";

const STORAGE_PREFIX = "superplane:github-install-started";
const MARKER_MAX_AGE_MS = 10 * 60_000;
const CHECK_DURATION_MS = 6_000;
const CHECK_POLL_INTERVAL_MS = 1_000;

interface InstallMarker {
  startedAt: number;
  knownKeys: string[];
}

interface InstallCheck {
  until: number;
  knownKeys: string[];
}

/** Each person tracks their own GitHub round trip per workspace. */
export interface GitHubInstallScope {
  accountId: string;
  factoryId: string;
}

function storageKey({ accountId, factoryId }: GitHubInstallScope): string | null {
  if (!accountId || !factoryId) return null;
  return `${STORAGE_PREFIX}:${accountId}:${factoryId}`;
}

/** Installations, repositories, and install requests that the user can already see. */
export function githubAccessKeys(data: MeDescribeVcsProviderOnboardingResponse | undefined): string[] | undefined {
  if (!data) return undefined;
  const repositories = (data.repositories ?? []).flatMap((repository) => [
    `installation:${repository.installationId}`,
    `repository:${repository.repositoryId}`,
  ]);
  const requests = (data.pendingRequests ?? []).map((request) => `request:${request.requestId}`);
  return [...new Set([...repositories, ...requests])];
}

export function markGitHubInstallStarted(scope: GitHubInstallScope, knownKeys: string[]): void {
  const key = storageKey(scope);
  if (!key) return;
  const marker: InstallMarker = { startedAt: Date.now(), knownKeys };
  localStorage.setItem(key, JSON.stringify(marker));
}

export function clearGitHubInstallStarted(scope: GitHubInstallScope): void {
  const key = storageKey(scope);
  if (key) localStorage.removeItem(key);
}

function readMarker(scope: GitHubInstallScope): InstallMarker | null {
  const key = storageKey(scope);
  if (!key) return null;
  try {
    const marker = JSON.parse(localStorage.getItem(key) ?? "null") as InstallMarker | null;
    if (!marker || Date.now() - marker.startedAt > MARKER_MAX_AGE_MS) return null;
    return marker;
  } catch {
    return null;
  }
}

function startCheck(scope: GitHubInstallScope): InstallCheck | null {
  const marker = readMarker(scope);
  if (!marker) return null;
  return { until: Date.now() + CHECK_DURATION_MS, knownKeys: marker.knownKeys };
}

/**
 * GitHub does not tell SuperPlane when a member sends an install request, so
 * the request shows only after the next catalog read. The first time a tab
 * opens or comes back after the user went to GitHub, it checks often for a
 * short time and reports that it is checking until new access arrives.
 */
export function useGitHubInstallReturn(
  scope: GitHubInstallScope,
  keys: string[] | undefined,
  refetch: () => unknown,
): boolean {
  const { accountId, factoryId } = scope;
  const [check, setCheck] = useState<InstallCheck | null>(() => startCheck(scope));

  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState !== "visible") return;
      const next = startCheck({ accountId, factoryId });
      if (next) setCheck(next);
    };
    onVisible();
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [accountId, factoryId]);

  useEffect(() => {
    if (check) clearGitHubInstallStarted({ accountId, factoryId });
  }, [check, accountId, factoryId]);

  const arrived = Boolean(check && keys?.some((key) => !check.knownKeys.includes(key)));

  useEffect(() => {
    if (!check) return;
    if (arrived) {
      setCheck(null);
      return;
    }
    const poll = window.setInterval(() => void refetch(), CHECK_POLL_INTERVAL_MS);
    const stop = window.setTimeout(() => setCheck(null), Math.max(0, check.until - Date.now()));
    return () => {
      window.clearInterval(poll);
      window.clearTimeout(stop);
    };
  }, [check, arrived, refetch, accountId, factoryId]);

  return Boolean(check) && !arrived;
}
