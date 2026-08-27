import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  pendingGitHubBrowserAction,
  pendingGitHubInstallPicker,
  startDirectGitHubConnect,
} from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("pendingGitHubBrowserAction", () => {
  it("returns the pending GitHub browser action", () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    expect(
      pendingGitHubBrowserAction(
        [
          {
            metadata: { integrationName: "github" },
            status: { state: "ready", browserAction: action, metadata: { startedByUserID: "user-1" } },
          },
          {
            metadata: { integrationName: "github" },
            status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
          },
        ],
        "user-1",
      ),
    ).toEqual(action);
  });

  it("returns undefined when every GitHub connection is ready", () => {
    expect(
      pendingGitHubBrowserAction([{ metadata: { integrationName: "github" }, status: { state: "ready" } }], "user-1"),
    ).toBeUndefined();
  });

  it("returns undefined when the pending row belongs to a teammate", () => {
    expect(
      pendingGitHubBrowserAction(
        [
          {
            metadata: { integrationName: "github" },
            status: {
              state: "pending",
              browserAction: { method: "GET", url: "https://github.com/apps/superplane/installations/new" },
              metadata: { startedByUserID: "user-1" },
            },
          },
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });
});

describe("pendingGitHubInstallPicker", () => {
  it("returns a pending GitHub connection with two or more installs", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
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
        ],
        "user-1",
      ),
    ).toEqual({ id: "int-1" });
  });

  it("returns undefined when there is no picker", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: { state: "pending", browserAction: { method: "GET", url: "https://github.com" } },
          },
        ],
        "user-1",
      ),
    ).toBeUndefined();
  });

  it("returns undefined when the picker belongs to a teammate", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
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
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });
});

describe("startDirectGitHubConnect", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("opens the picker page when pending installs exist", async () => {
    const create = vi.fn();
    const goTo = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/settings/integrations",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: {
              startedByUserID: "user-1",
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

    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/settings/integrations");
    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/int-1?setupStay=1");
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
          status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
        },
      ],
      currentUserId: "user-1",
      create,
    });

    expect(create).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/ws/setup?step=vcs&pick=newest");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a new connection when forceNew is set", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=3" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      existingNames: new Set(["github"]),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: { startedByUserID: "user-1" },
          },
        },
      ],
      currentUserId: "user-1",
      forceNew: true,
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "github", name: "github-2" });
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a new connection when the pending row belongs to a teammate", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=2" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      existingNames: new Set(["github"]),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: { startedByUserID: "user-1" },
          },
        },
      ],
      currentUserId: "user-2",
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "github", name: "github-2" });
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
