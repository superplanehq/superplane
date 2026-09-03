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

/**
 * Picks the organization slug to auto-redirect the account to, or `null`
 * when the account should stay on the organization picker.
 *
 * `organizations` must be keyed by slug (not UID) since the returned value
 * is used directly as the `/{slug}` URL segment.
 */
export function pickAutoRedirectOrganization(
  organizations: { slug: string }[],
  lastVisitedOrganizationSlug: string | null,
): string | null {
  if (organizations.length === 1) {
    return organizations[0].slug;
  }

  if (lastVisitedOrganizationSlug && organizations.some((org) => org.slug === lastVisitedOrganizationSlug)) {
    return lastVisitedOrganizationSlug;
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
