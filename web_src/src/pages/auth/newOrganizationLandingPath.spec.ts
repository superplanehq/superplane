import { describe, expect, it } from "vitest";

import { newOrganizationLandingPath } from "./newOrganizationLandingPath";

describe("newOrganizationLandingPath", () => {
  it("starts workspace setup in the new organization", () => {
    expect(newOrganizationLandingPath("org-1")).toBe("/org-1/workspaces/new");
  });

  it("falls back to the organization list when the id is missing", () => {
    expect(newOrganizationLandingPath("")).toBe("/");
  });
});
