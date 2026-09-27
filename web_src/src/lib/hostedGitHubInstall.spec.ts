import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  bindHostedGitHubInstallation,
  hostedGitHubAppSlug,
  hostedGitHubBindPath,
  hostedGitHubInstallRequested,
  hostedGitHubInstallRequestedAccount,
  hostedGitHubInstallURL,
  hostedGitHubStartedByLogin,
  hostedGitHubState,
  pendingGitHubInstallRequests,
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

  it("posts the selected installation and repository", async () => {
    const fetchSpy = stubFetch({ type: "basic", ok: true, status: 204 });

    await expect(bindHostedGitHubInstallation("csrf", "11", "22")).resolves.toBeUndefined();
    expect(fetchSpy).toHaveBeenCalledWith("/api/v1/github/app/bind", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: "state=csrf&installation_id=11&repository_id=22",
    });
  });

  it("accepts a plain success answer", async () => {
    stubFetch({ type: "basic", ok: true, status: 200 });

    await expect(bindHostedGitHubInstallation("csrf", "11", "22")).resolves.toBeUndefined();
  });

  it("throws on an error status", async () => {
    stubFetch({ type: "basic", ok: false, status: 404 });

    await expect(bindHostedGitHubInstallation("csrf", "11", "22")).rejects.toThrow(
      "Failed to connect the GitHub account",
    );
  });
});

describe("pendingGitHubInstallations", () => {
  it("reads valid rows", () => {
    expect(
      pendingGitHubInstallations({
        pendingInstallations: [
          {
            id: "11",
            accountLogin: "acme",
            accountType: "Organization",
            repositories: [{ id: 7, name: "acme/api", url: "https://github.com/acme/api" }],
          },
          { id: 22, accountLogin: "octo" },
        ],
      }),
    ).toEqual([
      {
        id: "11",
        accountLogin: "acme",
        accountType: "Organization",
        repositories: [{ id: "7", name: "acme/api", url: "https://github.com/acme/api" }],
      },
      { id: "22", accountLogin: "octo", repositories: [] },
    ]);
  });

  it("returns empty for missing or invalid metadata", () => {
    expect(pendingGitHubInstallations(undefined)).toEqual([]);
    expect(pendingGitHubInstallations({ pendingInstallations: [{ id: "", accountLogin: "acme" }] })).toEqual([]);
  });
});

describe("pendingGitHubInstallRequests", () => {
  it("reads every request and keeps legacy metadata compatible", () => {
    expect(
      pendingGitHubInstallRequests({
        installRequests: [
          { id: 1, accountLogin: "acme", requesterLogin: "member", createdAt: "2026-09-08T12:00:00Z" },
          { id: "2", accountLogin: "octo", requesterLogin: "member" },
        ],
      }),
    ).toEqual([
      { id: "1", accountLogin: "acme", requesterLogin: "member", createdAt: "2026-09-08T12:00:00Z" },
      { id: "2", accountLogin: "octo", requesterLogin: "member" },
    ]);
    expect(pendingGitHubInstallRequests({ installRequested: true, installRequestedAccount: "legacy" })).toEqual([
      { accountLogin: "legacy" },
    ]);
  });
});

describe("hosted GitHub URLs", () => {
  it("builds the public bind path", () => {
    expect(hostedGitHubBindPath()).toBe("/api/v1/github/app/bind");
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

  it("reads the GitHub login that authorized the connect", () => {
    expect(hostedGitHubStartedByLogin({ startedByGitHubLogin: "forestileao" })).toBe("forestileao");
    expect(hostedGitHubStartedByLogin({})).toBe("");
    expect(hostedGitHubStartedByLogin(undefined)).toBe("");
  });

  it("reads a pending GitHub install request", () => {
    expect(hostedGitHubInstallRequested({ installRequested: true })).toBe(true);
    expect(hostedGitHubInstallRequested({ installRequested: false })).toBe(false);
    expect(hostedGitHubInstallRequested({})).toBe(false);
    expect(hostedGitHubInstallRequested(undefined)).toBe(false);
  });

  it("reads the organization waiting for approval", () => {
    expect(hostedGitHubInstallRequestedAccount({ installRequestedAccount: "acme" })).toBe("acme");
    expect(hostedGitHubInstallRequestedAccount({ owner: "acme" })).toBe("");
    expect(hostedGitHubInstallRequestedAccount({ installRequestedAccount: "acme", owner: "other" })).toBe("acme");
    expect(hostedGitHubInstallRequestedAccount({})).toBe("");
  });
});
