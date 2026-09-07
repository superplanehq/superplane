import { afterEach, describe, expect, it, vi } from "vitest";

import {
  bindHostedGitHubInstallation,
  hostedGitHubAppSlug,
  hostedGitHubBindPath,
  hostedGitHubInstallRequested,
  hostedGitHubInstallRequestedAccount,
  hostedGitHubInstallURL,
  hostedGitHubState,
  pendingGitHubInstallations,
} from "./hostedGitHubInstall";

describe("bindHostedGitHubInstallation", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function stubFetch(response: Partial<Response>) {
    const fetchSpy = vi.fn().mockResolvedValue(response as Response);
    vi.stubGlobal("fetch", fetchSpy);
    return fetchSpy;
  }

  it("does not follow the success redirect, which can leave the page origin", async () => {
    const fetchSpy = stubFetch({ type: "opaqueredirect", ok: false, status: 0 });

    await expect(bindHostedGitHubInstallation("csrf", "11")).resolves.toBeUndefined();
    expect(fetchSpy).toHaveBeenCalledWith("/api/v1/github/app/bind?state=csrf&installation_id=11", {
      credentials: "same-origin",
      redirect: "manual",
    });
  });

  it("accepts a plain success answer", async () => {
    stubFetch({ type: "basic", ok: true, status: 200 });

    await expect(bindHostedGitHubInstallation("csrf", "11")).resolves.toBeUndefined();
  });

  it("throws on an error status", async () => {
    stubFetch({ type: "basic", ok: false, status: 404 });

    await expect(bindHostedGitHubInstallation("csrf", "11")).rejects.toThrow("Failed to connect the GitHub account");
  });
});

describe("pendingGitHubInstallations", () => {
  it("reads valid rows", () => {
    expect(
      pendingGitHubInstallations({
        pendingInstallations: [
          { id: "11", accountLogin: "acme", accountType: "Organization" },
          { id: 22, accountLogin: "octo" },
        ],
      }),
    ).toEqual([
      { id: "11", accountLogin: "acme", accountType: "Organization" },
      { id: "22", accountLogin: "octo" },
    ]);
  });

  it("returns empty for missing or invalid metadata", () => {
    expect(pendingGitHubInstallations(undefined)).toEqual([]);
    expect(pendingGitHubInstallations({ pendingInstallations: [{ id: "", accountLogin: "acme" }] })).toEqual([]);
  });
});

describe("hosted GitHub URLs", () => {
  it("builds the public bind path", () => {
    expect(hostedGitHubBindPath("csrf", "11")).toBe("/api/v1/github/app/bind?state=csrf&installation_id=11");
  });

  it("builds the GitHub install URL", () => {
    expect(hostedGitHubInstallURL("superplane", "csrf")).toBe(
      "https://github.com/apps/superplane/installations/new?state=csrf",
    );
  });

  it("reads state and slug", () => {
    expect(hostedGitHubState({ state: "csrf" })).toBe("csrf");
    expect(hostedGitHubAppSlug({ githubApp: { slug: "superplane" } })).toBe("superplane");
  });

  it("reads a pending GitHub install request", () => {
    expect(hostedGitHubInstallRequested({ installRequested: true })).toBe(true);
    expect(hostedGitHubInstallRequested({ installRequested: false })).toBe(false);
    expect(hostedGitHubInstallRequested({})).toBe(false);
    expect(hostedGitHubInstallRequested(undefined)).toBe(false);
  });

  it("reads the organization waiting for approval", () => {
    expect(hostedGitHubInstallRequestedAccount({ installRequestedAccount: "acme" })).toBe("acme");
    expect(hostedGitHubInstallRequestedAccount({ owner: "acme" })).toBe("acme");
    expect(hostedGitHubInstallRequestedAccount({ installRequestedAccount: "acme", owner: "other" })).toBe("acme");
    expect(hostedGitHubInstallRequestedAccount({})).toBe("");
  });
});
