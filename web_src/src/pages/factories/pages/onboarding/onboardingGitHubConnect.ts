import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";

const STORAGE_PREFIX = "superplane:onboarding-github-connect";
const CONNECTED_PARAM = "githubConnected";

const returnedVisits = new Set<string>();
let releaseReturnTimer: ReturnType<typeof setTimeout> | undefined;

function accountStorageKey(accountId: string): string {
  return `${STORAGE_PREFIX}:${accountId}`;
}

function workspaceStorageKey(accountId: string, factoryId: string): string {
  return `${STORAGE_PREFIX}:${accountId}:${factoryId}`;
}

function visitKey(accountId: string, factoryId: string): string | null {
  if (!accountId || !factoryId) return null;
  return `${accountId}:${factoryId}`;
}

function noteGitHubConnectReturn(accountId: string, factoryId: string): void {
  const key = visitKey(accountId, factoryId);
  if (!key) return;
  if (releaseReturnTimer) {
    clearTimeout(releaseReturnTimer);
    releaseReturnTimer = undefined;
  }
  returnedVisits.add(key);
}

function githubConnectReturnNoted(accountId: string, factoryId: string): boolean {
  const key = visitKey(accountId, factoryId);
  return Boolean(key && returnedVisits.has(key));
}

function scheduleGitHubConnectReturnRelease(): void {
  if (releaseReturnTimer) clearTimeout(releaseReturnTimer);
  releaseReturnTimer = setTimeout(() => {
    clearGitHubConnectReturnVisit();
  }, 0);
}

/** Drops a GitHub return that belongs to this page visit, including a pending release. */
export function clearGitHubConnectReturnVisit(): void {
  if (releaseReturnTimer) {
    clearTimeout(releaseReturnTimer);
    releaseReturnTimer = undefined;
  }
  returnedVisits.clear();
}

function legacyConnectionForAccount(accountId: string): boolean {
  const prefix = `${STORAGE_PREFIX}:${accountId}:`;
  for (let index = 0; index < localStorage.length; index += 1) {
    const key = localStorage.key(index);
    if (key?.startsWith(prefix) && localStorage.getItem(key) === "yes") return true;
  }
  return false;
}

export function readOnboardingGitHubConnect(accountId: string, factoryId: string): boolean {
  if (!accountId) return false;
  if (localStorage.getItem(accountStorageKey(accountId)) === "yes") return true;
  if (factoryId && localStorage.getItem(workspaceStorageKey(accountId, factoryId)) === "yes") return true;
  return legacyConnectionForAccount(accountId);
}

export function writeOnboardingGitHubConnect(accountId: string): void {
  if (!accountId) return;
  localStorage.setItem(accountStorageKey(accountId), "yes");
}

/** GitHub sends the user to this path only after the connection succeeds. */
export function githubConnectReturnPath(path: string): string {
  const [pathname, search = ""] = path.split("?");
  const searchParams = new URLSearchParams(search);
  searchParams.set(CONNECTED_PARAM, "1");
  return `${pathname}?${searchParams.toString()}`;
}

/**
 * Records the GitHub connect redirect in this browser. Repository selection
 * uses the linked GitHub identity from the server, so this record does not
 * decide whether GitHub is connected.
 *
 * A connection saved before that change is stored per workspace. Any of those
 * records for this account still counts.
 *
 * A saved repository proves an earlier connection on this workspace. That proof
 * stays for the session when the user clears the repository to choose another one.
 */
export function useOnboardingGitHubConnect(args: { accountId: string; factoryId: string; connectedBefore: boolean }): {
  connected: boolean;
  returned: boolean;
} {
  const { accountId, factoryId, connectedBefore } = args;
  const [searchParams, setSearchParams] = useSearchParams();
  const returnedFromGitHub = searchParams.get(CONNECTED_PARAM) === "1";
  const [keptConnection, setKeptConnection] = useState(connectedBefore);
  if (connectedBefore && !keptConnection) setKeptConnection(true);
  if (returnedFromGitHub) noteGitHubConnectReturn(accountId, factoryId);
  const returned = returnedFromGitHub || githubConnectReturnNoted(accountId, factoryId);

  useEffect(() => {
    if (returned) noteGitHubConnectReturn(accountId, factoryId);
    return () => scheduleGitHubConnectReturnRelease();
  }, [returned, accountId, factoryId]);

  useEffect(() => {
    if (!returnedFromGitHub || !accountId) return;
    writeOnboardingGitHubConnect(accountId);
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete(CONNECTED_PARAM);
        return next;
      },
      { replace: true },
    );
  }, [returnedFromGitHub, accountId, setSearchParams]);

  return {
    connected: returnedFromGitHub || keptConnection || readOnboardingGitHubConnect(accountId, factoryId),
    returned,
  };
}
