const STORAGE_PREFIX = "integration-setup-return";
const MAX_AGE_MS = 60 * 60 * 1000;

interface StoredReturn {
  path: string;
  createdAt: number;
}

function storageKey(organizationId: string, integrationId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}:${integrationId}`;
}

function isSafePath(path: string, organizationId: string): boolean {
  return path.startsWith(`/${organizationId}/`) && !path.startsWith("//");
}

export function rememberIntegrationSetupReturn(
  organizationId: string,
  integrationId: string,
  path: string | undefined,
): void {
  if (!integrationId || !path || !isSafePath(path, organizationId)) return;

  const value: StoredReturn = { path, createdAt: Date.now() };
  window.localStorage.setItem(storageKey(organizationId, integrationId), JSON.stringify(value));
}

export function peekIntegrationSetupReturn(organizationId: string, integrationId: string): string | null {
  if (!integrationId) return null;

  const key = storageKey(organizationId, integrationId);
  const raw = window.localStorage.getItem(key);
  if (!raw) return null;

  try {
    const value = JSON.parse(raw) as Partial<StoredReturn>;
    if (
      typeof value.path !== "string" ||
      typeof value.createdAt !== "number" ||
      !isSafePath(value.path, organizationId) ||
      Date.now() - value.createdAt > MAX_AGE_MS
    ) {
      window.localStorage.removeItem(key);
      return null;
    }
    return value.path;
  } catch {
    window.localStorage.removeItem(key);
    return null;
  }
}

export function consumeIntegrationSetupReturn(organizationId: string, integrationId: string): void {
  window.localStorage.removeItem(storageKey(organizationId, integrationId));
}
