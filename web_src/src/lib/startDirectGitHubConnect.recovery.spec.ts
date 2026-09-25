import { beforeEach, describe, expect, it, vi } from "bun:test";

import { startDirectGitHubConnect } from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
  isOnboardingSetupReturnPath: (path?: string) => path?.split("?")[0] === "/onboarding",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("startDirectGitHubConnect recovery", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("retries an existing failed connection without creating a duplicate", async () => {
    const failed = {
      metadata: { id: "int-1", integrationName: "github" },
      status: {
        state: "error",
        stateDescription: "GitHub was temporarily unavailable",
        metadata: { startedByUserID: "user-1" },
      },
    };
    const recovered = {
      ...failed,
      status: {
        state: "pending",
        metadata: {
          startedByUserID: "user-1",
          state: "csrf",
          githubApp: { slug: "superplane" },
          pendingInstallations: [{ id: "11", accountLogin: "acme", repositories: [{ id: 101, name: "acme/api" }] }],
        },
      },
    };
    const create = vi.fn();
    const update = vi.fn().mockResolvedValue(recovered);

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(["github"]),
      connected: [failed],
      currentUserId: "user-1",
      create,
      update,
    });

    expect(started).toBe(false);
    expect(create).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      id: "int-1",
      configuration: { setupReturnPath: "/onboarding?attempt=1&step=vcs" },
    });
    expect(follow).not.toHaveBeenCalled();
  });

  it("keeps an existing failed connection when its retry fails", async () => {
    const create = vi.fn();
    const update = vi.fn().mockResolvedValue(undefined);

    await expect(
      startDirectGitHubConnect({
        organizationId: "org-1",
        returnTo: "/onboarding?attempt=1&step=vcs",
        existingNames: new Set(["github"]),
        connected: [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "error",
              stateDescription: "GitHub was temporarily unavailable",
              metadata: { startedByUserID: "user-1" },
            },
          },
        ],
        currentUserId: "user-1",
        create,
        update,
      }),
    ).rejects.toThrow("GitHub was temporarily unavailable");

    expect(create).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledTimes(1);
    expect(follow).not.toHaveBeenCalled();
  });

  it("reports a retry that has no next action", async () => {
    const create = vi.fn();
    const update = vi.fn().mockResolvedValue({
      metadata: { id: "int-1", integrationName: "github" },
      status: { state: "pending", metadata: { startedByUserID: "user-1" } },
    });

    await expect(
      startDirectGitHubConnect({
        organizationId: "org-1",
        returnTo: "/onboarding?attempt=1&step=vcs",
        existingNames: new Set(["github"]),
        connected: [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: { state: "error", metadata: { startedByUserID: "user-1" } },
          },
        ],
        currentUserId: "user-1",
        create,
        update,
      }),
    ).rejects.toThrow("SuperPlane could not connect to GitHub. Try again.");

    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
  });
});
