import { describe, expect, it } from "vitest";

import {
  organizationMatchesRoute,
  organizationRouteId,
  parseAccountOrganizations,
  readyAccountOrganizations,
  selectedOrganizationRouteId,
} from "./accountOrganizations";

describe("parseAccountOrganizations", () => {
  it("keeps organizations that have an id and a name, including slug when present", () => {
    expect(
      parseAccountOrganizations([
        {
          id: "org-uuid",
          slug: "demo",
          name: "Demo",
          lastLocationPath: "/demo/apps/deploy",
          lastLocationUpdatedAt: "2026-09-07T12:00:00.000Z",
        },
        { id: "org-1", name: "SuperPlane" },
        { id: 2, name: "skip-me" },
        { name: "missing-id" },
      ]),
    ).toEqual([
      {
        id: "org-uuid",
        slug: "demo",
        name: "Demo",
        lastLocationPath: "/demo/apps/deploy",
        lastLocationUpdatedAt: "2026-09-07T12:00:00.000Z",
      },
      { id: "org-1", name: "SuperPlane" },
    ]);
  });

  it("returns an empty list when the body is not an array", () => {
    expect(parseAccountOrganizations({ id: "org-1" })).toEqual([]);
  });

  it("keeps the unfinished initial onboarding flag", () => {
    expect(parseAccountOrganizations([{ id: "org-pending", name: "Pending", initialOnboardingPending: true }])).toEqual(
      [{ id: "org-pending", name: "Pending", initialOnboardingPending: true }],
    );
  });
});

describe("readyAccountOrganizations", () => {
  it("hides organizations that did not finish first-run setup", () => {
    expect(
      readyAccountOrganizations([
        { id: "org-ready", name: "Ready" },
        { id: "org-pending", name: "Pending", initialOnboardingPending: true },
      ]),
    ).toEqual([{ id: "org-ready", name: "Ready" }]);
  });
});

describe("organizationRouteId", () => {
  it("prefers the slug used in URLs", () => {
    expect(organizationRouteId({ id: "org-uuid", slug: "demo" })).toBe("demo");
  });

  it("falls back to the id when the slug is missing", () => {
    expect(organizationRouteId({ id: "org-uuid" })).toBe("org-uuid");
  });
});

describe("organizationMatchesRoute", () => {
  const demo = { id: "org-uuid", slug: "demo" };

  it("matches the current organization when the URL uses the slug", () => {
    expect(organizationMatchesRoute(demo, "demo")).toBe(true);
  });

  it("matches the current organization when the URL uses the id", () => {
    expect(organizationMatchesRoute(demo, "org-uuid")).toBe(true);
  });

  it("does not match a different organization", () => {
    expect(organizationMatchesRoute(demo, "acme")).toBe(false);
  });
});

describe("selectedOrganizationRouteId", () => {
  const organizations = [
    { id: "org-uuid", slug: "demo" },
    { id: "org-acme", slug: "acme" },
  ];

  it("returns the slug when the route uses the organization id", () => {
    expect(selectedOrganizationRouteId(organizations, "org-uuid")).toBe("demo");
  });

  it("returns the slug when the route already uses the slug", () => {
    expect(selectedOrganizationRouteId(organizations, "demo")).toBe("demo");
  });

  it("returns the route id when no organization matches", () => {
    expect(selectedOrganizationRouteId(organizations, "unknown")).toBe("unknown");
  });
});
