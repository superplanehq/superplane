const STORAGE_PREFIX = "superplane:onboarding-jira-project";

function storageKey(factoryId: string, integrationId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}:${integrationId}`;
}

export function readOnboardingJiraProject(factoryId: string, integrationId: string): string {
  if (!factoryId || !integrationId) return "";
  return localStorage.getItem(storageKey(factoryId, integrationId)) ?? "";
}

export function writeOnboardingJiraProject(factoryId: string, integrationId: string, projectId: string): void {
  if (!factoryId || !integrationId) return;
  if (!projectId) {
    localStorage.removeItem(storageKey(factoryId, integrationId));
    return;
  }
  localStorage.setItem(storageKey(factoryId, integrationId), projectId);
}
