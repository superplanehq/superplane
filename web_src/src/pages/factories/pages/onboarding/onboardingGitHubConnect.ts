import { useEffect, useState } from "react";
import { useSearchParams } from "react-router";

const STORAGE_PREFIX = "superplane:onboarding-github-connect";
const CONNECTED_PARAM = "githubConnected";

function storageKey(accountId: string, factoryId: string): string {
  return `${STORAGE_PREFIX}:${accountId}:${factoryId}`;
}

export function readOnboardingGitHubConnect(accountId: string, factoryId: string): boolean {
  if (!accountId || !factoryId) return false;
  return localStorage.getItem(storageKey(accountId, factoryId)) === "yes";
}

export function writeOnboardingGitHubConnect(accountId: string, factoryId: string): void {
  if (!accountId || !factoryId) return;
  localStorage.setItem(storageKey(accountId, factoryId), "yes");
}

/** GitHub sends the user to this path only after the connection succeeds. */
export function githubConnectReturnPath(path: string): string {
  const [pathname, search = ""] = path.split("?");
  const searchParams = new URLSearchParams(search);
  searchParams.set(CONNECTED_PARAM, "1");
  return `${pathname}?${searchParams.toString()}`;
}

/**
 * GitHub sign-in links a GitHub identity to the account. Workspace setup must
 * not use that identity until the user connects GitHub, so each person's
 * completed connection is saved per workspace.
 *
 * A saved repository proves an earlier connection. That proof stays for the
 * session when the user clears the repository to choose another one.
 */
export function useOnboardingGitHubConnect(args: {
  accountId: string;
  factoryId: string;
  connectedBefore: boolean;
}): boolean {
  const { accountId, factoryId, connectedBefore } = args;
  const [searchParams, setSearchParams] = useSearchParams();
  const returnedFromGitHub = searchParams.get(CONNECTED_PARAM) === "1";
  const [keptConnection, setKeptConnection] = useState(connectedBefore);
  if (connectedBefore && !keptConnection) setKeptConnection(true);

  useEffect(() => {
    if (!returnedFromGitHub || !accountId) return;
    writeOnboardingGitHubConnect(accountId, factoryId);
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete(CONNECTED_PARAM);
        return next;
      },
      { replace: true },
    );
  }, [returnedFromGitHub, accountId, factoryId, setSearchParams]);

  return returnedFromGitHub || keptConnection || readOnboardingGitHubConnect(accountId, factoryId);
}
