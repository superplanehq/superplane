const STORAGE_PREFIX = "superplane:onboarding-repo";

function storageKey(factoryId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}`;
}

/** The repository chosen before onboarding saves it. A reload reads this value. */
export function readOnboardingRepoChoice(factoryId: string): string | null {
  if (!factoryId) return null;
  const value = sessionStorage.getItem(storageKey(factoryId));
  if (!value?.trim()) return null;
  return value;
}

export function writeOnboardingRepoChoice(factoryId: string, repo: string): void {
  if (!factoryId || !repo.trim()) return;
  sessionStorage.setItem(storageKey(factoryId), repo);
}

export function clearOnboardingRepoChoice(factoryId: string): void {
  if (!factoryId) return;
  sessionStorage.removeItem(storageKey(factoryId));
}
