import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";

const STORAGE_PREFIX = "superplane:bitbucket-install-attempt";

export interface BitbucketInstallationAttempt {
  startedAt: number;
  knownSlugs: string[];
}

function storageKey(organizationId: string): string | null {
  if (!organizationId) return null;
  return `${STORAGE_PREFIX}:${organizationId}`;
}

function read(organizationId: string): BitbucketInstallationAttempt | null {
  const key = storageKey(organizationId);
  if (!key) return null;
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return null;
    const attempt = JSON.parse(raw) as Partial<BitbucketInstallationAttempt> | null;
    if (
      !attempt ||
      typeof attempt.startedAt !== "number" ||
      !Array.isArray(attempt.knownSlugs) ||
      Date.now() - attempt.startedAt > bitbucketInstallationAttemptMs
    ) {
      localStorage.removeItem(key);
      return null;
    }
    return {
      startedAt: attempt.startedAt,
      knownSlugs: attempt.knownSlugs.filter((slug): slug is string => typeof slug === "string"),
    };
  } catch {
    return null;
  }
}

export function restoreBitbucketInstallAttempt(organizationId: string): BitbucketInstallationAttempt | null {
  return read(organizationId);
}

export function markBitbucketInstallAttempt(organizationId: string, knownSlugs: string[]): void {
  const key = storageKey(organizationId);
  if (!key) return;
  const attempt: BitbucketInstallationAttempt = { startedAt: Date.now(), knownSlugs };
  try {
    localStorage.setItem(key, JSON.stringify(attempt));
  } catch {
    // Blocked storage must not block onboarding.
  }
}

export function clearBitbucketInstallAttempt(organizationId: string): void {
  const key = storageKey(organizationId);
  if (key) localStorage.removeItem(key);
}
