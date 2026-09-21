import { beforeEach, describe, expect, it, vi } from "bun:test";

import { findPendingJiraConnection, findReadyJiraConnection, startDirectJiraConnect } from "./startDirectJiraConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_RETURN_PATH_KEY: "setupReturnPath",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("findReadyJiraConnection", () => {
  it("returns the first ready Jira connection", () => {
    const ready = findReadyJiraConnection([
      { metadata: { integrationName: "github", id: "gh-1" }, status: { state: "ready" } },
      { metadata: { integrationName: "jira", id: "jira-1" }, status: { state: "ready" } },
    ]);
    expect(ready?.metadata?.id).toBe("jira-1");
  });
});

describe("findPendingJiraConnection", () => {
  it("returns a pending Jira connection with a browser action", () => {
    const pending = findPendingJiraConnection([
      { metadata: { integrationName: "jira", id: "jira-1" }, status: { state: "pending" } },
      {
        metadata: { integrationName: "jira", id: "jira-2" },
        status: { state: "pending", browserAction: { method: "GET", url: "https://auth.atlassian.com/authorize" } },
      },
    ]);
    expect(pending?.metadata?.id).toBe("jira-2");
  });
});

describe("startDirectJiraConnect", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("creates a connection and opens Atlassian authorization", async () => {
    const action = { method: "GET", url: "https://auth.atlassian.com/authorize?client_id=abc" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    const started = await startDirectJiraConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/sp/lines/line-1/setup/jira",
      existingNames: new Set(["jira"]),
      create,
    });

    expect(started).toBe(true);
    expect(create).toHaveBeenCalledWith({
      integrationName: "jira",
      name: "jira-2",
      configuration: { setupReturnPath: "/org-1/workspaces/sp/lines/line-1/setup/jira" },
    });
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/sp/lines/line-1/setup/jira");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("reuses a ready Jira connection instead of creating another one", async () => {
    const create = vi.fn();
    const onExistingReady = vi.fn();

    const started = await startDirectJiraConnect({
      organizationId: "org-1",
      existingNames: new Set(["jira"]),
      connected: [{ metadata: { integrationName: "jira", id: "jira-1" }, status: { state: "ready" } }],
      onExistingReady,
      create,
    });

    expect(started).toBe(false);
    expect(create).not.toHaveBeenCalled();
    expect(onExistingReady).toHaveBeenCalledWith("jira-1");
    expect(follow).not.toHaveBeenCalled();
  });

  it("resumes pending Jira authorization instead of creating another connection", async () => {
    const action = { method: "GET", url: "https://auth.atlassian.com/authorize?state=abc" };
    const create = vi.fn();

    const started = await startDirectJiraConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/sp/lines/line-1/setup/jira",
      existingNames: new Set(["jira"]),
      connected: [
        {
          metadata: { integrationName: "jira", id: "jira-1", name: "jira" },
          status: { state: "pending", browserAction: action },
        },
      ],
      create,
    });

    expect(started).toBe(true);
    expect(create).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/sp/lines/line-1/setup/jira");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("omits setupReturnPath when the caller has no return path", async () => {
    const action = { method: "GET", url: "https://auth.atlassian.com/authorize" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectJiraConnect({
      organizationId: "org-1",
      existingNames: new Set(),
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "jira", name: "jira" });
    expect(remember).toHaveBeenCalledWith("org-1", undefined);
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("throws when create does not return a browser action", async () => {
    const create = vi.fn().mockResolvedValue({
      integration: { status: {} },
    });

    await expect(
      startDirectJiraConnect({
        organizationId: "org-1",
        existingNames: new Set(),
        create,
      }),
    ).rejects.toThrow("The Jira authorization page did not open.");
    expect(follow).not.toHaveBeenCalled();
  });
});
