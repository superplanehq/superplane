const STORAGE_PREFIX = "integration-setup-return";
const MAX_AGE_MS = 15 * 60 * 1000;

interface StoredReturn {
  path: string;
  createdAt: number;
}

// Keyed by organization only, not by integration id: the legacy GitHub connect
// creates a new integration during the round trip to the provider, so the id the
// caller knows before leaving does not match the id the provider redirects to.
function storageKey(organizationId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}`;
}

function isSafePath(path: string, organizationId: string): boolean {
  return path.startsWith(`/${organizationId}/`) && !path.startsWith("//");
}

export function rememberIntegrationSetupReturn(organizationId: string, path: string | undefined): void {
  if (!organizationId || !path || !isSafePath(path, organizationId)) return;

  const value: StoredReturn = { path, createdAt: Date.now() };
  window.localStorage.setItem(storageKey(organizationId), JSON.stringify(value));
}

export function peekIntegrationSetupReturn(organizationId: string): string | null {
  if (!organizationId) return null;

  const key = storageKey(organizationId);
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

export function consumeIntegrationSetupReturn(organizationId: string): void {
  window.localStorage.removeItem(storageKey(organizationId));
}
