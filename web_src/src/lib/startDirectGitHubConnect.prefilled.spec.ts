import { beforeEach, describe, expect, it, vi } from "bun:test";

import { startDirectGitHubConnect } from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

// Identity discovery can prefill the account picker on create. Such a
// connection carries no browser action, so the connect stays in the app and
// shows the picker instead of opening GitHub.
describe("startDirectGitHubConnect with a prefilled picker", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("stays on onboarding when create returns a prefilled picker", async () => {
    const create = vi.fn().mockResolvedValue({
      integration: {
        metadata: { id: "int-9", integrationName: "github" },
        status: {
          state: "pending",
          metadata: {
            startedByUserID: "user-1",
            state: "csrf",
            githubApp: { slug: "superplane" },
            pendingInstallations: [{ id: "11", accountLogin: "acme" }],
          },
        },
      },
    });

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(true);
    expect(create).toHaveBeenCalledTimes(1);
    expect(follow).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/onboarding?attempt=1&step=vcs");
  });

  it("opens the picker page when create returns a prefilled picker outside onboarding", async () => {
    const goTo = vi.fn();
    const create = vi.fn().mockResolvedValue({
      integration: {
        metadata: { id: "int-9", integrationName: "github" },
        status: {
          state: "pending",
          metadata: {
            startedByUserID: "user-1",
            pendingInstallations: [
              { id: "11", accountLogin: "acme" },
              { id: "22", accountLogin: "octo" },
            ],
          },
        },
      },
    });

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/settings/integrations",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      create,
      goTo,
    });

    expect(started).toBe(true);
    expect(follow).not.toHaveBeenCalled();
    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/int-9?setupStay=1");
  });

  it("follows the browser action when the created picker is empty", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=1" };
    const create = vi.fn().mockResolvedValue({
      integration: {
        metadata: { id: "int-9", integrationName: "github" },
        status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
      },
    });

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(true);
    expect(follow).toHaveBeenCalledWith(action);
  });
});
