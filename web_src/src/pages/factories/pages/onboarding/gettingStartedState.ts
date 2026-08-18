const STORAGE_PREFIX = "superplane:workspace-getting-started";

function storageKey(organizationId: string, factoryId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}:${factoryId}`;
}

export function markWorkspaceGettingStarted(organizationId: string, factoryId: string): void {
  localStorage.setItem(storageKey(organizationId, factoryId), "true");
}

export function shouldShowWorkspaceGettingStarted(organizationId: string, factoryId: string): boolean {
  return localStorage.getItem(storageKey(organizationId, factoryId)) === "true";
}

export function dismissWorkspaceGettingStarted(organizationId: string, factoryId: string): void {
  localStorage.removeItem(storageKey(organizationId, factoryId));
}
