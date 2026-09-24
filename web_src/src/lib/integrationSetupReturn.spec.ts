import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  INTEGRATION_SETUP_RETURN_COOKIE,
  consumeIntegrationSetupReturn,
  consumeIntegrationSetupReturnIfArrived,
  hasGitHubSetupRequest,
  hasIntegrationSetupStay,
  peekIntegrationSetupReturnPreferredIntegration,
  peekIntegrationSetupReturn,
  rememberIntegrationSetupReturn,
  withGitHubSetupRequest,
  configurationWithSetupReturnPath,
} from "./integrationSetupReturn";

function setupReturnCookie(): string | undefined {
  const prefix = `${INTEGRATION_SETUP_RETURN_COOKIE}=`;
  const value = document.cookie
    .split("; ")
    .find((part) => part.startsWith(prefix))
    ?.slice(prefix.length);
  return value === undefined || value === "" ? undefined : value;
}

describe("integration setup return", () => {
  afterEach(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
    document.cookie = `${INTEGRATION_SETUP_RETURN_COOKIE}=; Path=/; Max-Age=0`;
    vi.useRealTimers();
  });

  it("stores and consumes a return path for an organization", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup");

    consumeIntegrationSetupReturn("org-1");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
    expect(setupReturnCookie()).toBeUndefined();
  });

  it("stores the integration that started the provider round trip", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup?step=agent", "openrouter-1");

    expect(peekIntegrationSetupReturnPreferredIntegration("org-1")).toBe("openrouter-1");

    consumeIntegrationSetupReturn("org-1");
    expect(peekIntegrationSetupReturnPreferredIntegration("org-1")).toBeNull();
  });

  it("keeps a provider return isolated from another tab", () => {
    const providerPath = "/org-1/workspaces/APP/setup?step=agent";
    rememberIntegrationSetupReturn("org-1", providerPath, "openrouter-1");

    // A legacy setup in another tab writes organization-wide local storage.
    window.localStorage.setItem(
      "integration-setup-return:org-1",
      JSON.stringify({ path: "/org-1/workspaces/OTHER/setup", createdAt: Date.now() }),
    );

    expect(peekIntegrationSetupReturn("org-1")).toBe(providerPath);
    expect(peekIntegrationSetupReturnPreferredIntegration("org-1")).toBe("openrouter-1");
  });

  it("mirrors the return path in a cookie for the GitHub callback", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup?step=vcs&pick=newest");

    expect(decodeURIComponent(setupReturnCookie() ?? "")).toBe("/org-1/workspaces/APP/setup?step=vcs&pick=newest");
  });

  it("consumes the marker only after the browser arrives on the stored page", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup?step=vcs&pick=newest");

    consumeIntegrationSetupReturnIfArrived("org-1", "/org-1/settings/integrations/abc");
    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup?step=vcs&pick=newest");

    consumeIntegrationSetupReturnIfArrived("org-1", "/org-1/workspaces/APP/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });

  it("detects the setupStay query", () => {
    expect(hasIntegrationSetupStay("setupStay=1")).toBe(true);
    expect(hasIntegrationSetupStay("?setupStay=1")).toBe(true);
    expect(hasIntegrationSetupStay("")).toBe(false);
  });

  it("copies a GitHub install request onto the stored return path", () => {
    expect(hasGitHubSetupRequest("githubSetup=request")).toBe(true);
    expect(hasGitHubSetupRequest("?githubSetup=request")).toBe(true);
    expect(hasGitHubSetupRequest("")).toBe(false);
    expect(withGitHubSetupRequest("/org-1/workspaces/APP/setup?step=vcs", "")).toBe(
      "/org-1/workspaces/APP/setup?step=vcs",
    );
    expect(withGitHubSetupRequest("/org-1/workspaces/APP/setup?step=vcs", "githubSetup=request")).toBe(
      "/org-1/workspaces/APP/setup?step=vcs&githubSetup=request",
    );
    expect(withGitHubSetupRequest("/org-1/workspaces/APP/setup?step=vcs", "githubSetup=request&githubOrg=acme")).toBe(
      "/org-1/workspaces/APP/setup?step=vcs&githubSetup=request&githubOrg=acme",
    );
    expect(
      withGitHubSetupRequest(
        "/org-1/workspaces/APP/setup?step=vcs",
        "githubSetup=request&githubOrg=acme&githubIntegrationId=int-1",
      ),
    ).toBe("/org-1/workspaces/APP/setup?step=vcs&githubSetup=request&githubOrg=acme&githubIntegrationId=int-1");
  });

  it("returns the path regardless of the integration the provider redirects to", () => {
    // The legacy connect creates a new integration id during the round trip, so
    // the marker must not depend on any specific integration id.
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup");
  });

  it("rejects paths outside the organization", () => {
    rememberIntegrationSetupReturn("org-1", "/org-2/workspaces/APP/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();

    rememberIntegrationSetupReturn("org-1", "https://example.com");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });

  it("accepts the account onboarding route as an integration return", () => {
    rememberIntegrationSetupReturn("org-1", "/onboarding?attempt=attempt-1&step=vcs&pick=newest");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/onboarding?attempt=attempt-1&step=vcs&pick=newest");
  });

  it("adds setupReturnPath when a return path is present", () => {
    expect(
      configurationWithSetupReturnPath({ clientId: "id" }, "/org-1/workspaces/sp/lines/line-1/setup/jira"),
    ).toEqual({
      clientId: "id",
      setupReturnPath: "/org-1/workspaces/sp/lines/line-1/setup/jira",
    });
    expect(configurationWithSetupReturnPath({ clientId: "id" })).toEqual({ clientId: "id" });
  });

  it("expires a return path after fifteen minutes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-19T12:00:00Z"));
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    vi.setSystemTime(new Date("2026-08-19T12:15:01Z"));
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });
});
