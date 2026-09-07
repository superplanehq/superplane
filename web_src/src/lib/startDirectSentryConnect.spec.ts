import { beforeEach, describe, expect, it, vi } from "vitest";

import { pendingSentryBrowserAction, startDirectSentryConnect } from "./startDirectSentryConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("pendingSentryBrowserAction", () => {
  it("returns the pending Sentry browser action", () => {
    const action = { method: "GET", url: "https://app.example/api/v1/sentry/app/start" };
    expect(
      pendingSentryBrowserAction(
        [
          {
            metadata: { integrationName: "sentry" },
            status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
          },
        ],
        "user-1",
      ),
    ).toEqual(action);
  });

  it("returns undefined when the pending row belongs to a teammate", () => {
    expect(
      pendingSentryBrowserAction(
        [
          {
            metadata: { integrationName: "sentry" },
            status: {
              state: "pending",
              browserAction: { method: "GET", url: "https://app.example/api/v1/sentry/app/start" },
              metadata: { startedByUserID: "user-1" },
            },
          },
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });
});

describe("startDirectSentryConnect", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("creates a Sentry connection and follows the browser action", async () => {
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: { method: "GET", url: "https://app.example/api/v1/sentry/app/start" } } },
    });

    const started = await startDirectSentryConnect({
      organizationId: "org-1",
      returnTo: "/onboarding",
      existingNames: new Set(),
      connected: [],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(true);
    expect(create).toHaveBeenCalledWith({
      integrationName: "sentry",
      name: "sentry",
      configuration: { setupReturnPath: "/onboarding" },
    });
    expect(follow).toHaveBeenCalledWith({ method: "GET", url: "https://app.example/api/v1/sentry/app/start" });
  });

  it("resumes a pending Sentry browser action", async () => {
    const create = vi.fn();
    const action = { method: "GET", url: "https://app.example/api/v1/sentry/app/start" };

    await startDirectSentryConnect({
      organizationId: "org-1",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "sentry" },
          status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
        },
      ],
      currentUserId: "user-1",
      create,
    });

    expect(create).not.toHaveBeenCalled();
    expect(follow).toHaveBeenCalledWith(action);
  });
});
