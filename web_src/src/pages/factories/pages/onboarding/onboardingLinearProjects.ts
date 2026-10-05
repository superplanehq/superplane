const STORAGE_PREFIX = "superplane:onboarding-linear-projects";

function storageKey(factoryId: string, integrationId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}:${integrationId}`;
}

export function readOnboardingLinearProjects(factoryId: string, integrationId: string): string[] {
  if (!factoryId || !integrationId) return [];
  const raw = localStorage.getItem(storageKey(factoryId, integrationId));
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((id): id is string => typeof id === "string" && id.length > 0);
  } catch {
    return [];
  }
}

export function writeOnboardingLinearProjects(factoryId: string, integrationId: string, projectIds: string[]): void {
  if (!factoryId || !integrationId) return;
  if (projectIds.length === 0) {
    localStorage.removeItem(storageKey(factoryId, integrationId));
    return;
  }
  localStorage.setItem(storageKey(factoryId, integrationId), JSON.stringify(projectIds));
}
