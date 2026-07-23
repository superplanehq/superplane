import { describe, expect, it } from "vitest";
import type { RolesRole } from "@/api-client/types.gen";
import { getAssignableApiKeyRoles } from "./apiKeyRoles";

const role = (name: string, displayName?: string): RolesRole => ({
  metadata: { name },
  spec: { displayName },
});

describe("getAssignableApiKeyRoles", () => {
  it("includes custom roles so they appear in the API key dropdown", () => {
    const roles = [role("org_admin", "Admin"), role("deployer", "Deployer")];

    const result = getAssignableApiKeyRoles(roles).map((r) => r.metadata?.name);

    expect(result).toContain("deployer");
  });

  it("lists custom roles first, then built-in roles", () => {
    const roles = [
      role("org_viewer", "Viewer"),
      role("org_admin", "Admin"),
      role("deployer", "Deployer"),
      role("auditor", "Auditor"),
    ];

    const result = getAssignableApiKeyRoles(roles).map((r) => r.metadata?.name);

    // Custom roles (alphabetical) first, then built-ins (alphabetical).
    expect(result).toEqual(["auditor", "deployer", "org_admin", "org_viewer"]);
  });

  it("excludes org_owner from the assignable roles", () => {
    const roles = [role("org_owner", "Owner"), role("org_admin", "Admin"), role("org_viewer", "Viewer")];

    const result = getAssignableApiKeyRoles(roles).map((r) => r.metadata?.name);

    expect(result).not.toContain("org_owner");
    expect(result).toEqual(["org_admin", "org_viewer"]);
  });

  it("falls back to role name when a display name is missing", () => {
    const roles = [role("zeta"), role("alpha")];

    const result = getAssignableApiKeyRoles(roles).map((r) => r.metadata?.name);

    expect(result).toEqual(["alpha", "zeta"]);
  });

  it("returns an empty list when there are no roles", () => {
    expect(getAssignableApiKeyRoles([])).toEqual([]);
  });
});
