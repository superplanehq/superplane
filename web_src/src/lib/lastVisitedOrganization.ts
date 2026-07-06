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
    for (const [accountId, organizationId] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof organizationId === "string" && organizationId) {
        result[accountId] = organizationId;
      }
    }

    return result;
  } catch {
    return {};
  }
}

export function readLastVisitedOrganization(accountId: string): string | null {
  if (!accountId) {
    return null;
  }

  return readAllLastVisitedOrganizations()[accountId] ?? null;
}

export function pickAutoRedirectOrganization(
  organizations: { id: string }[],
  lastVisitedOrganizationId: string | null,
): string | null {
  if (organizations.length === 1) {
    return organizations[0].id;
  }

  if (lastVisitedOrganizationId && organizations.some((org) => org.id === lastVisitedOrganizationId)) {
    return lastVisitedOrganizationId;
  }

  return null;
}

export function recordLastVisitedOrganization(accountId: string, organizationId: string): void {
  if (!accountId || !organizationId || typeof window === "undefined") {
    return;
  }

  try {
    const all = readAllLastVisitedOrganizations();
    all[accountId] = organizationId;
    window.localStorage.setItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Last-visited persistence is optional.
  }
}
