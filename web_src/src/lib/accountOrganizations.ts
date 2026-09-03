export interface AccountOrganization {
  id: string;
  name: string;
  slug?: string;
}

export function parseAccountOrganizations(body: unknown): AccountOrganization[] {
  if (!Array.isArray(body)) {
    return [];
  }

  return body.flatMap((entry) => {
    const organization = parseAccountOrganization(entry);
    return organization ? [organization] : [];
  });
}

export function organizationRouteId(organization: Pick<AccountOrganization, "id" | "slug">): string {
  return organization.slug || organization.id;
}

export function organizationMatchesRoute(
  organization: Pick<AccountOrganization, "id" | "slug">,
  routeId: string,
): boolean {
  return organization.id === routeId || organization.slug === routeId;
}

export function selectedOrganizationRouteId(
  organizations: Array<Pick<AccountOrganization, "id" | "slug">>,
  routeId: string,
): string {
  const selected = organizations.find((organization) => organizationMatchesRoute(organization, routeId));
  return selected ? organizationRouteId(selected) : routeId;
}

function parseAccountOrganization(entry: unknown): AccountOrganization | null {
  if (!entry || typeof entry !== "object") {
    return null;
  }

  const candidate = entry as Record<string, unknown>;
  if (typeof candidate.id !== "string" || typeof candidate.name !== "string") {
    return null;
  }

  return {
    id: candidate.id,
    name: candidate.name,
    ...(typeof candidate.slug === "string" && candidate.slug ? { slug: candidate.slug } : {}),
  };
}
