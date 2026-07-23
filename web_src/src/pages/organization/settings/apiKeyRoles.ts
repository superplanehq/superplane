import type { RolesRole } from "@/api-client/types.gen";

// Built-in organization roles that may be assigned to an API key / service
// account. org_owner is intentionally excluded because it is reserved for
// human users and must never be assignable to an API key to avoid privilege
// escalation.
const ASSIGNABLE_BUILTIN_ROLES = new Set(["org_admin", "org_viewer"]);

// Roles that must never be offered when creating an API key, regardless of
// whether they are built-in or custom.
const RESERVED_ROLES = new Set(["org_owner"]);

const isBuiltinRole = (name: string) => ASSIGNABLE_BUILTIN_ROLES.has(name);

const byDisplayName = (a: RolesRole, b: RolesRole) =>
  (a.spec?.displayName || a.metadata?.name || "").localeCompare(b.spec?.displayName || b.metadata?.name || "");

/**
 * Returns the roles that can be assigned to an API key, sorted so that custom
 * roles appear first (alphabetically by display name), followed by the
 * assignable built-in roles. Reserved roles such as org_owner are excluded.
 *
 * This mirrors the backend validation in create_api_key.go, which accepts any
 * organization role except org_owner.
 */
export function getAssignableApiKeyRoles(roles: RolesRole[]): RolesRole[] {
  const selectable = roles.filter((role) => !RESERVED_ROLES.has(role.metadata?.name || ""));

  const customRoles = selectable.filter((role) => !isBuiltinRole(role.metadata?.name || "")).sort(byDisplayName);
  const builtinRoles = selectable.filter((role) => isBuiltinRole(role.metadata?.name || "")).sort(byDisplayName);

  return [...customRoles, ...builtinRoles];
}
