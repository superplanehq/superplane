import { describe, expect, it } from "vitest";

import { resolveOrganizationUidRedirect } from "./organizationPath";

describe("resolveOrganizationUidRedirect", () => {
  it("redirects a root URL that uses the org UID to the org slug", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/org-uid-123",
        segment: "org-uid-123",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBe("/acme");
  });

  it("redirects a nested URL that uses the org UID to the org slug", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/org-uid-123/settings/general",
        segment: "org-uid-123",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBe("/acme/settings/general");
  });

  it("preserves the query string and hash during the swap", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/org-uid-123/canvas/123",
        search: "?run=abc",
        hash: "#node-1",
        segment: "org-uid-123",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBe("/acme/canvas/123?run=abc#node-1");
  });

  it("does nothing when the segment already matches the slug", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/acme/settings",
        segment: "acme",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBeNull();
  });

  it("does nothing when the segment is not the org's UID (unrelated slug)", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/some-other-org/settings",
        segment: "some-other-org",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBeNull();
  });

  it("does nothing when the organization has no slug yet", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/org-uid-123",
        segment: "org-uid-123",
        organizationId: "org-uid-123",
        organizationSlug: "",
      }),
    ).toBeNull();
  });

  it("does nothing when the segment or org id is missing", () => {
    expect(
      resolveOrganizationUidRedirect({
        pathname: "/",
        segment: "",
        organizationId: "org-uid-123",
        organizationSlug: "acme",
      }),
    ).toBeNull();

    expect(
      resolveOrganizationUidRedirect({
        pathname: "/org-uid-123",
        segment: "org-uid-123",
        organizationId: "",
        organizationSlug: "acme",
      }),
    ).toBeNull();
  });
});
