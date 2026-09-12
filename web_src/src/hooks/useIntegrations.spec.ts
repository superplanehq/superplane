import { beforeEach, describe, expect, it, vi } from "bun:test";
const { organizationsListIntegrationResources } = vi.hoisted(() => ({
  organizationsListIntegrationResources: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsListIntegrationResources,
}));

import { resolveGithubDefaultBranch } from "./useIntegrations";

describe("resolveGithubDefaultBranch", () => {
  beforeEach(() => {
    organizationsListIntegrationResources.mockReset();
  });

  it("returns the branch reported by the integration", async () => {
    organizationsListIntegrationResources.mockResolvedValue({
      data: { resources: [{ type: "default_branch", name: "staging", id: "staging" }] },
    });

    const branch = await resolveGithubDefaultBranch("org-1", "int-1", "acme/app");

    expect(branch).toBe("staging");
    expect(organizationsListIntegrationResources).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1", integrationId: "int-1" },
        query: { type: "default_branch", repository: "acme/app" },
      }),
    );
  });

  it("falls back to main when the integration id is missing", async () => {
    const branch = await resolveGithubDefaultBranch("org-1", "", "acme/app");

    expect(branch).toBe("main");
    expect(organizationsListIntegrationResources).not.toHaveBeenCalled();
  });

  it("falls back to main when the repository is missing", async () => {
    const branch = await resolveGithubDefaultBranch("org-1", "int-1", "");

    expect(branch).toBe("main");
    expect(organizationsListIntegrationResources).not.toHaveBeenCalled();
  });

  it("falls back to main when the response has no resources", async () => {
    organizationsListIntegrationResources.mockResolvedValue({ data: { resources: [] } });

    const branch = await resolveGithubDefaultBranch("org-1", "int-1", "acme/app");

    expect(branch).toBe("main");
  });
});
