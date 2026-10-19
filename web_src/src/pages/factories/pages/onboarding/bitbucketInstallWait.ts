const STORAGE_PREFIX = "superplane:bitbucket-install-wait";

export type BitbucketInstallWait = {
  startedAt: number;
  knownSlugs: string[];
};

function storageKey(organizationId: string): string | null {
  const id = organizationId.trim();
  if (!id) return null;
  return `${STORAGE_PREFIX}:${id}`;
}

function isInstallWait(value: unknown): value is BitbucketInstallWait {
  if (!value || typeof value !== "object") return false;
  const record = value as { startedAt?: unknown; knownSlugs?: unknown };
  return (
    typeof record.startedAt === "number" &&
    Number.isFinite(record.startedAt) &&
    Array.isArray(record.knownSlugs) &&
    record.knownSlugs.every((slug) => typeof slug === "string")
  );
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
    // A full or blocked store must not stop the in-memory wait.
  }
}

function removeStoredValue(key: string): void {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // A blocked store must not stop the in-memory wait from ending.
  }
}

/** The Bitbucket install wait for this browser tab. A refresh reads this value. */
export function readBitbucketInstallWait(organizationId: string): BitbucketInstallWait | null {
  const key = storageKey(organizationId);
  if (!key) return null;
  const raw = readStoredValue(key);
  if (!raw) return null;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!isInstallWait(parsed)) return null;
    return { startedAt: parsed.startedAt, knownSlugs: [...parsed.knownSlugs] };
  } catch {
    return null;
  }
}

export function writeBitbucketInstallWait(organizationId: string, wait: BitbucketInstallWait): void {
  const key = storageKey(organizationId);
  if (!key || !isInstallWait(wait)) return;
  writeStoredValue(key, JSON.stringify({ startedAt: wait.startedAt, knownSlugs: [...wait.knownSlugs] }));
}

export function clearBitbucketInstallWait(organizationId: string): void {
  const key = storageKey(organizationId);
  if (!key) return;
  removeStoredValue(key);
}
