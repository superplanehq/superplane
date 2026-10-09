const STORAGE_PREFIX = "superplane:onboarding-repo";

function storageKey(factoryId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}`;
}

function readStoredValue(key: string): string | null {
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStoredValue(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // A full or blocked store must not stop the in-memory selection.
  }
}

function removeStoredValue(key: string): void {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // A blocked store must not stop a clear or a host change.
  }
}

/** The repository chosen before onboarding saves it. A reload reads this value. */
export function readOnboardingRepoChoice(factoryId: string): string | null {
  if (!factoryId) return null;
  const value = readStoredValue(storageKey(factoryId));
  if (!value?.trim()) return null;
  return value;
}

export function writeOnboardingRepoChoice(factoryId: string, repo: string): void {
  if (!factoryId || !repo.trim()) return;
  writeStoredValue(storageKey(factoryId), repo);
}

export function clearOnboardingRepoChoice(factoryId: string): void {
  if (!factoryId) return;
  removeStoredValue(storageKey(factoryId));
}
