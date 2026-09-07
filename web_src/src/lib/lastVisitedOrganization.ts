import { isSafeRedirectPath, pathBelongsToOrganization } from "./safeRedirectPath";

// Organizations are addressed by slug in the URL (e.g. `/{slug}/...`), so
// every value stored and read here is expected to be an organization slug,
// never its UID. Callers are responsible for passing slugs; see
// `pickAutoRedirectOrganization` and `recordLastVisitedOrganization` below.
export const LAST_VISITED_ORGANIZATION_STORAGE_KEY = "superplane:last-visited-organization";

type LastVisitedOrganizationByAccount = Record<string, string>;

function readAllLastVisitedOrganizations(): LastVisitedOrganizationByAccount {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }

    const result: LastVisitedOrganizationByAccount = {};
    for (const [accountId, organizationSlug] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof organizationSlug === "string" && organizationSlug) {
        result[accountId] = organizationSlug;
      }
    }

    return result;
  } catch {
    return {};
  }
}

/** Returns the last-visited organization slug for the account, if any. */
export function readLastVisitedOrganization(accountId: string): string | null {
  if (!accountId) {
    return null;
  }

  return readAllLastVisitedOrganizations()[accountId] ?? null;
}

export type AutoRedirectOrganization = {
  slug: string;
  lastLocationUpdatedAt?: string | null;
};

/**
 * Picks the organization slug to auto-redirect the account to, or `null`
 * when the account has no organizations.
 *
 * Prefer the last visited slug when the account still belongs to it. Else
 * prefer the organization with the newest saved screen (cross-device). Else
 * the first organization (callers pass newest-first).
 */
export function pickAutoRedirectOrganization(
  organizations: AutoRedirectOrganization[],
  lastVisitedOrganizationSlug: string | null,
): string | null {
  if (organizations.length === 0) {
    return null;
  }

  if (lastVisitedOrganizationSlug && organizations.some((org) => org.slug === lastVisitedOrganizationSlug)) {
    return lastVisitedOrganizationSlug;
  }

  let latestSlug: string | null = null;
  let latestAt = "";
  for (const organization of organizations) {
    const updatedAt = organization.lastLocationUpdatedAt ?? "";
    if (updatedAt && updatedAt > latestAt) {
      latestAt = updatedAt;
      latestSlug = organization.slug;
    }
  }
  if (latestSlug) {
    return latestSlug;
  }

  return organizations[0].slug;
}

/** First safe path that belongs to this organization, or null. */
export function pickResumePath(
  organizationSlug: string,
  ...candidates: Array<string | null | undefined>
): string | null {
  for (const path of candidates) {
    if (path && isSafeRedirectPath(path) && pathBelongsToOrganization(path, organizationSlug)) {
      return path;
    }
  }
  return null;
}

/** Records the organization slug the account last visited. */
export function recordLastVisitedOrganization(accountId: string, organizationSlug: string): void {
  if (!accountId || !organizationSlug || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAllLastVisitedOrganizations();
    all[accountId] = organizationSlug;
    window.localStorage.setItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}
