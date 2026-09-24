export const DEFAULT_ORGANIZATION_ROLE_NAMES = ["org_admin", "org_maintainer", "org_operator"] as const;

export const DEFAULT_ORGANIZATION_ROLE_ORDER = ["org_admin", "org_maintainer", "org_operator"] as const;

export type DefaultOrganizationRoleName = (typeof DEFAULT_ORGANIZATION_ROLE_NAMES)[number];

export function isDefaultOrganizationRole(roleName: string | undefined | null): boolean {
  if (!roleName) {
    return false;
  }

  return (DEFAULT_ORGANIZATION_ROLE_NAMES as readonly string[]).includes(roleName);
}

export function defaultOrganizationRoleSortIndex(roleName: string | undefined | null): number {
  if (!roleName) {
    return Number.MAX_SAFE_INTEGER;
  }

  const index = (DEFAULT_ORGANIZATION_ROLE_ORDER as readonly string[]).indexOf(roleName);
  return index === -1 ? Number.MAX_SAFE_INTEGER : index;
}
