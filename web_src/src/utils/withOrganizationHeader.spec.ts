import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

describe("withOrganizationHeader", () => {
  const setPathname = (pathname: string) => {
    window.history.pushState({}, "", pathname);
  };

  beforeEach(() => {
    setPathname("/");
  });

  afterEach(() => {
    setPathname("/");
  });

  it("adds x-organization-id based on window.location.pathname by default", () => {
    setPathname("/org-from-url/canvases");

    const options = withOrganizationHeader();
    expect(options.headers["x-organization-id"]).toBe("org-from-url");
  });

  it("prefers explicit organizationId when window.location is stale", () => {
    // Regression test: this is the scenario that caused a transient 404 on first navigation.
    // During router transitions, window.location.pathname can still point at the previous route.
    setPathname("/old-org-id");

    const options = withOrganizationHeader({ organizationId: "new-org-id" });
    expect(options.headers["x-organization-id"]).toBe("new-org-id");
  });

  it("does not leak organizationId into the returned options", () => {
    setPathname("/old-org-id");

    const options = withOrganizationHeader({ organizationId: "new-org-id" });
    expect(options.organizationId).toBeUndefined();
  });

  it("merges provided headers and preserves them", () => {
    setPathname("/old-org-id");

    const options = withOrganizationHeader({
      organizationId: "new-org-id",
      headers: {
        "content-type": "application/json",
      },
    });

    expect(options.headers["content-type"]).toBe("application/json");
    expect(options.headers["x-organization-id"]).toBe("new-org-id");
  });
});
