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
// connection carries no browser action, so the connect shows the picker
// instead of opening GitHub.
describe("startDirectGitHubConnect with a prefilled picker", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("reloads the onboarding picker when create returns a prefilled picker", async () => {
    const assign = vi.fn();
    const previousAssign = window.location.assign;
    window.location.assign = assign as unknown as typeof window.location.assign;
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

    try {
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
      expect(assign).toHaveBeenCalledWith("/onboarding?attempt=1&step=vcs");
      expect(remember).toHaveBeenCalledWith("org-1", "/onboarding?attempt=1&step=vcs");
    } finally {
      window.location.assign = previousAssign;
    }
  });

  // Identity discovery prefills the picker, so onboarding shows its own
  // account picker. The connect reloads the setup step instead of opening
  // the GitHub install page.
  it("reloads the onboarding picker when stored options exist", async () => {
    const create = vi.fn();
    const goTo = vi.fn();
    const assign = vi.fn();
    const previousAssign = window.location.assign;
    window.location.assign = assign as unknown as typeof window.location.assign;

    try {
      const started = await startDirectGitHubConnect({
        organizationId: "org-1",
        returnTo: "/onboarding?attempt=1&step=vcs",
        existingNames: new Set(),
        connected: [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                state: "csrf",
                githubApp: { slug: "superplane" },
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        currentUserId: "user-1",
        create,
        goTo,
      });

      expect(started).toBe(true);
      expect(create).not.toHaveBeenCalled();
      expect(follow).not.toHaveBeenCalled();
      expect(goTo).not.toHaveBeenCalled();
      expect(assign).toHaveBeenCalledWith("/onboarding?attempt=1&step=vcs");
      expect(remember).toHaveBeenCalledWith("org-1", "/onboarding?attempt=1&step=vcs");
    } finally {
      window.location.assign = previousAssign;
    }
  });

  // A connection without a stored bind state cannot bind its options, so
  // the click starts a fresh connect.
  it("starts a fresh connect on onboarding when the picker kept no bind state", async () => {
    const action = { method: "GET", url: "https://github.com/login/oauth/authorize?client_id=new" };
    const create = vi.fn().mockResolvedValue({ integration: { status: { browserAction: action } } });

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [{ id: "11", accountLogin: "acme" }],
            },
          },
        },
      ],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(true);
    expect(create).toHaveBeenCalledTimes(1);
    expect(follow).toHaveBeenCalledWith(action);
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
