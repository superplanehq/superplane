import { useCallback, useState } from "react";

const STORAGE_PREFIX = "superplane:onboarding-github-connect";

function storageKey(factoryId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}`;
}

export function readOnboardingGitHubConnect(factoryId: string): boolean {
  if (!factoryId) return false;
  return localStorage.getItem(storageKey(factoryId)) === "yes";
}

export function writeOnboardingGitHubConnect(factoryId: string): void {
  if (!factoryId) return;
  localStorage.setItem(storageKey(factoryId), "yes");
}

/**
 * GitHub sign-in links a GitHub identity to the account. Workspace setup must
 * not use that identity until the user clicks Connect GitHub, so the click is
 * saved per workspace and survives the GitHub redirect.
 */
export function useOnboardingGitHubConnect(factoryId: string) {
  const [confirmed, setConfirmed] = useState(() => readOnboardingGitHubConnect(factoryId));
  const confirm = useCallback(() => {
    writeOnboardingGitHubConnect(factoryId);
    setConfirmed(true);
  }, [factoryId]);
  return [confirmed, confirm] as const;
}
