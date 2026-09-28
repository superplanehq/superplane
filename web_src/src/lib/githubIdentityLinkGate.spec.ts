import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  githubConnectResumeError,
  githubIdentityLinkPath,
  githubSwitchAccountPath,
  hasGitHubIdentity,
  isGitHubConnectResumeReturn,
  redirectToGitHubIdentityLink,
  stripGitHubConnectResumeParams,
  withGitHubConnectResumeMarker,
} from "./githubIdentityLinkGate";

describe("hasGitHubIdentity", () => {
  it("accepts a linked GitHub account", () => {
    expect(hasGitHubIdentity({ linked_accounts: [{ provider: "github", username: "octocat" }] })).toBe(true);
  });

  it("accepts a GitHub sign-in method", () => {
    expect(hasGitHubIdentity({ providers: [{ provider: "github" }] })).toBe(true);
  });

  it("rejects an account without a GitHub identity", () => {
    expect(hasGitHubIdentity({ providers: [{ provider: "google" }], linked_accounts: [] })).toBe(false);
    expect(hasGitHubIdentity(undefined)).toBe(false);
  });

  it("rejects a linked account without a username", () => {
    expect(hasGitHubIdentity({ linked_accounts: [{ provider: "github" }] })).toBe(false);
  });
});

describe("resume marker", () => {
  it("adds the marker and keeps existing query params", () => {
    expect(withGitHubConnectResumeMarker("/onboarding?attempt=1")).toBe("/onboarding?attempt=1&githubConnect=resume");
  });

  it("builds the link path with the encoded return path", () => {
    expect(githubIdentityLinkPath("/onboarding")).toBe(
      "/auth/github?intent=connect&redirect=%2Fonboarding%3FgithubConnect%3Dresume",
    );
  });

  it("builds the switch path with the provider account picker forced", () => {
    expect(githubSwitchAccountPath("/onboarding?step=vcs")).toBe(
      "/auth/github?intent=connect&select_account=1&redirect=%2Fonboarding%3Fstep%3Dvcs%26githubConnect%3Dresume",
    );
  });

  it("detects the return from the link flow only with an auth result", () => {
    expect(isGitHubConnectResumeReturn("?githubConnect=resume&linked_account=linked&provider=github")).toBe(true);
    expect(isGitHubConnectResumeReturn("?githubConnect=resume&auth_error=linked_account_in_use")).toBe(true);
    expect(isGitHubConnectResumeReturn("?githubConnect=resume")).toBe(false);
    expect(isGitHubConnectResumeReturn("?linked_account=linked")).toBe(false);
  });

  it("reads the auth error", () => {
    expect(githubConnectResumeError("?auth_error=linked_account_in_use")).toBe("linked_account_in_use");
    expect(githubConnectResumeError("?linked_account=linked")).toBe("");
  });

  it("strips the auth result params and keeps the rest", () => {
    expect(
      stripGitHubConnectResumeParams("?attempt=1&githubConnect=resume&linked_account=linked&provider=github"),
    ).toBe("attempt=1");
  });
});

describe("redirectToGitHubIdentityLink", () => {
  const assign = vi.fn();
  let previousAssign: typeof window.location.assign;
  let previousFetch: typeof globalThis.fetch;

  beforeEach(() => {
    assign.mockClear();
    previousAssign = window.location.assign;
    window.location.assign = assign as unknown as typeof window.location.assign;
    previousFetch = globalThis.fetch;
  });

  afterEach(() => {
    window.location.assign = previousAssign;
    globalThis.fetch = previousFetch;
  });

  function stubFetch(config: unknown, account: unknown) {
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      const body = path.includes("/auth/config") ? config : account;
      return new Response(JSON.stringify(body), { status: 200 });
    }) as unknown as typeof globalThis.fetch;
  }

  it("redirects to the link flow when the account has no GitHub identity", async () => {
    stubFetch({ providers: ["github"] }, { providers: [{ provider: "password" }], linked_accounts: [] });

    const redirected = await redirectToGitHubIdentityLink("/onboarding");

    expect(redirected).toBe(true);
    expect(assign).toHaveBeenCalledWith("/auth/github?intent=connect&redirect=%2Fonboarding%3FgithubConnect%3Dresume");
  });

  it("skips the gate when the identity is already known", async () => {
    stubFetch({ providers: ["github"] }, { linked_accounts: [{ provider: "github", username: "octocat" }] });

    expect(await redirectToGitHubIdentityLink("/onboarding")).toBe(false);
    expect(assign).not.toHaveBeenCalled();
  });

  it("skips the gate when GitHub sign-in is not configured", async () => {
    stubFetch({ providers: ["google"] }, { linked_accounts: [] });

    expect(await redirectToGitHubIdentityLink("/onboarding")).toBe(false);
    expect(assign).not.toHaveBeenCalled();
  });

  it("skips the gate when the lookups fail", async () => {
    globalThis.fetch = vi.fn(async () => {
      throw new Error("offline");
    }) as unknown as typeof globalThis.fetch;

    expect(await redirectToGitHubIdentityLink("/onboarding")).toBe(false);
    expect(assign).not.toHaveBeenCalled();
  });
});
