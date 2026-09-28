import { beforeEach, describe, expect, it, vi } from "bun:test";

import { startDirectGitHubConnect } from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));
const redirectToLink = vi.hoisted(() => vi.fn(async () => false));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

vi.mock("@/lib/githubIdentityLinkGate", () => ({
  redirectToGitHubIdentityLink: redirectToLink,
}));

// The identity gate runs before any integration is created. When it starts
// the link flow, the connect stops; the resume after the link skips the gate.
describe("startDirectGitHubConnect identity gate", () => {
  const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=1" };
  const createdWithAction = {
    integration: {
      metadata: { id: "int-1", integrationName: "github" },
      status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
    },
  };

  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
    redirectToLink.mockClear();
    redirectToLink.mockResolvedValue(false);
  });

  it("stops at the link flow before creating an integration", async () => {
    redirectToLink.mockResolvedValue(true);
    const create = vi.fn();

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(true);
    expect(redirectToLink).toHaveBeenCalledWith("/onboarding?attempt=1&step=vcs");
    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
  });

  it("continues the connect when the gate passes", async () => {
    const create = vi.fn().mockResolvedValue(createdWithAction);

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
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("skips the gate on the resume after the link flow", async () => {
    const create = vi.fn().mockResolvedValue(createdWithAction);

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      skipIdentityGate: true,
      create,
    });

    expect(started).toBe(true);
    expect(redirectToLink).not.toHaveBeenCalled();
    expect(follow).toHaveBeenCalledWith(action);
  });
});
