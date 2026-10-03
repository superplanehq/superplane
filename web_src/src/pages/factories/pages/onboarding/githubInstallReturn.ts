import type { MeDescribeVcsProviderOnboardingResponse } from "@/api-client";
import { useEffect, useState } from "react";

const STORAGE_KEY = "superplane:github-install-started";
const MARKER_MAX_AGE_MS = 10 * 60_000;
const CHECK_DURATION_MS = 15_000;
const CHECK_POLL_INTERVAL_MS = 1_000;

interface InstallMarker {
  startedAt: number;
  knownKeys: string[];
}

/** Installations and install requests that the user can already see. */
export function githubAccessKeys(data: MeDescribeVcsProviderOnboardingResponse | undefined): string[] | undefined {
  if (!data) return undefined;
  const installations = (data.repositories ?? []).map((repository) => `installation:${repository.installationId}`);
  const requests = (data.pendingRequests ?? []).map((request) => `request:${request.requestId}`);
  return [...new Set([...installations, ...requests])];
}

export function markGitHubInstallStarted(knownKeys: string[]): void {
  const marker: InstallMarker = { startedAt: Date.now(), knownKeys };
  localStorage.setItem(STORAGE_KEY, JSON.stringify(marker));
}

function readMarker(): InstallMarker | null {
  try {
    const marker = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null") as InstallMarker | null;
    if (!marker || Date.now() - marker.startedAt > MARKER_MAX_AGE_MS) return null;
    return marker;
  } catch {
    return null;
  }
}

/**
 * GitHub does not tell SuperPlane when a member sends an install request, so
 * the request shows only after the next catalog read. A tab that opens or
 * comes back after the user went to GitHub checks often for a short time and
 * reports that it is checking until new access arrives.
 */
export function useGitHubInstallReturn(keys: string[] | undefined, refetch: () => unknown): boolean {
  const [check, setCheck] = useState<{ until: number; knownKeys: string[] } | null>(() => startCheck());

  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState !== "visible") return;
      const next = startCheck();
      if (next) setCheck(next);
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, []);

  const arrived = Boolean(check && keys?.some((key) => !check.knownKeys.includes(key)));

  useEffect(() => {
    if (!check) return;
    if (arrived) {
      localStorage.removeItem(STORAGE_KEY);
      setCheck(null);
      return;
    }
    const poll = window.setInterval(() => void refetch(), CHECK_POLL_INTERVAL_MS);
    const stop = window.setTimeout(() => setCheck(null), Math.max(0, check.until - Date.now()));
    return () => {
      window.clearInterval(poll);
      window.clearTimeout(stop);
    };
  }, [check, arrived, refetch]);

  return Boolean(check) && !arrived;
}

function startCheck(): { until: number; knownKeys: string[] } | null {
  const marker = readMarker();
  if (!marker) return null;
  return { until: Date.now() + CHECK_DURATION_MS, knownKeys: marker.knownKeys };
}
