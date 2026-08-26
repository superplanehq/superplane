import { beforeEach, describe, expect, it, vi } from "vitest";

import { pendingGitHubBrowserAction, startDirectGitHubConnect } from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("pendingGitHubBrowserAction", () => {
  it("returns the pending GitHub browser action", () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    expect(
      pendingGitHubBrowserAction([
        {
          metadata: { integrationName: "github" },
          status: { state: "ready", browserAction: action },
        },
        {
          metadata: { integrationName: "github" },
          status: { state: "pending", browserAction: action },
        },
      ]),
    ).toEqual(action);
  });

  it("returns undefined when every GitHub connection is ready", () => {
    expect(
      pendingGitHubBrowserAction([{ metadata: { integrationName: "github" }, status: { state: "ready" } }]),
    ).toBeUndefined();
  });
});

describe("startDirectGitHubConnect", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("reuses a pending browser action", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    const create = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs&pick=newest",
      existingNames: new Set(),
      connected: [
        {
          metadata: { integrationName: "github" },
          status: { state: "pending", browserAction: action },
        },
      ],
      create,
    });

    expect(create).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/ws/setup?step=vcs&pick=newest");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a connection and follows the new browser action", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=1" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/settings/integrations",
      existingNames: new Set(["github"]),
      connected: [],
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "github", name: "github-2" });
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/settings/integrations");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("throws when create does not return a browser action", async () => {
    const create = vi.fn().mockResolvedValue({
      integration: { status: { setupState: { currentStep: { name: "selectOwner" } } } },
    });

    await expect(
      startDirectGitHubConnect({
        organizationId: "org-1",
        existingNames: new Set(),
        connected: [],
        create,
      }),
    ).rejects.toThrow("The GitHub App install page did not open.");
  });
});
