import { beforeEach, describe, expect, it, vi } from "bun:test";

import { startDirectJiraConnect } from "./startDirectJiraConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_RETURN_PATH_KEY: "setupReturnPath",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

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
