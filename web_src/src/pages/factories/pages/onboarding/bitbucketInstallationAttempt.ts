const STORAGE_PREFIX = "superplane:onboarding-bitbucket-install-attempt";

/** One Bitbucket installation attempt. A reload restores it for the same organization. */
export type BitbucketInstallationAttempt = {
  startedAt: number;
  knownSlugs: string[];
};

function storageKey(organizationId: string): string {
  return `${STORAGE_PREFIX}:${organizationId}`;
}

function isAttempt(value: unknown): value is BitbucketInstallationAttempt {
  if (typeof value !== "object" || value === null) return false;
  const { startedAt, knownSlugs } = value as Partial<BitbucketInstallationAttempt>;
  return (
    typeof startedAt === "number" &&
    Number.isFinite(startedAt) &&
    Array.isArray(knownSlugs) &&
    knownSlugs.every((slug) => typeof slug === "string")
  );
}

export function readBitbucketInstallationAttempt(organizationId: string): BitbucketInstallationAttempt | null {
  if (!organizationId) return null;
  try {
    const raw = sessionStorage.getItem(storageKey(organizationId));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    return isAttempt(parsed) ? { startedAt: parsed.startedAt, knownSlugs: parsed.knownSlugs } : null;
  } catch {
    return null;
  }
}

export function writeBitbucketInstallationAttempt(organizationId: string, attempt: BitbucketInstallationAttempt): void {
  if (!organizationId) return;
  try {
    sessionStorage.setItem(storageKey(organizationId), JSON.stringify(attempt));
  } catch {
    // A full or blocked store must not stop the in-memory attempt.
  }
}

export function clearBitbucketInstallationAttempt(organizationId: string): void {
  if (!organizationId) return;
  try {
    sessionStorage.removeItem(storageKey(organizationId));
  } catch {
    // A blocked store must not stop the attempt from ending.
  }
}
