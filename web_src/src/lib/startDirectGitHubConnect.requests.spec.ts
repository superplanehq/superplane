import { beforeEach, describe, expect, it, vi } from "vitest";

import { startDirectGitHubConnect } from "./startDirectGitHubConnect";

const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: vi.fn(),
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));
vi.mock("@/lib/browserAction", () => ({ followBrowserAction: follow }));

describe("startDirectGitHubConnect request selection", () => {
  beforeEach(() => follow.mockClear());

  it("reuses the exact requested connection when the user has two pending connections", async () => {
    const firstAction = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=1" };
    const requestedAction = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=2" };
    const create = vi.fn();
    const update = vi.fn().mockResolvedValue(undefined);

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: { state: "pending", browserAction: firstAction, metadata: { startedByUserID: "user-1" } },
        },
        {
          metadata: { id: "int-2", integrationName: "github" },
          status: { state: "pending", browserAction: requestedAction, metadata: { startedByUserID: "user-1" } },
        },
      ],
      currentUserId: "user-1",
      preferredIntegrationId: "int-2",
      create,
      update,
    });

    expect(create).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      id: "int-2",
      configuration: { setupReturnPath: "/org-1/workspaces/ws/setup?step=vcs" },
    });
    expect(follow).toHaveBeenCalledWith(requestedAction);
  });

  it("reuses an exact ready connection that has another installation request", async () => {
    const authorizeAction = "https://github.com/login/oauth/authorize?state=ready";
    const create = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-ready", integrationName: "github" },
          status: {
            state: "ready",
            metadata: {
              startedByUserID: "user-1",
              state: "csrf",
              authorizeURL: authorizeAction,
              installRequested: true,
              pendingInstallations: [{ id: "11", accountLogin: "acme" }],
            },
          },
        },
      ],
      currentUserId: "user-1",
      preferredIntegrationId: "int-ready",
      create,
    });

    expect(create).not.toHaveBeenCalled();
    expect(follow).toHaveBeenCalledWith({ method: "GET", url: authorizeAction });
  });
});
