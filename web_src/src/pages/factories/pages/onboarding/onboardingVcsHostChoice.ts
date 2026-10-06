import type { VcsHostId } from "./onboardingFixtures";

const STORAGE_PREFIX = "superplane:onboarding-vcs-host";

function storageKey(factoryId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}`;
}

function isRememberedHost(value: string | null): value is "github" | "bitbucket" {
  return value === "github" || value === "bitbucket";
}

/** The host chosen before a repository is saved. A reload reads this value. */
export function readOnboardingVcsHostChoice(factoryId: string): VcsHostId | null {
  if (!factoryId) return null;
  const value = sessionStorage.getItem(storageKey(factoryId));
  return isRememberedHost(value) ? value : null;
}

export function writeOnboardingVcsHostChoice(factoryId: string, host: VcsHostId): void {
  if (!factoryId || !isRememberedHost(host)) return;
  sessionStorage.setItem(storageKey(factoryId), host);
}
