import { describe, expect, it } from "vitest";

import {
  hostedGitHubAppSlug,
  hostedGitHubBindPath,
  hostedGitHubInstallRequested,
  hostedGitHubInstallRequestedAccount,
  hostedGitHubInstallURL,
  hostedGitHubState,
  pendingGitHubInstallations,
  sortGitHubInstallations,
} from "./hostedGitHubInstall";

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
    expect(hostedGitHubInstallURL("superplane", "")).toBe("https://github.com/apps/superplane/installations/new");
    expect(hostedGitHubInstallURL("superplane", "csrf", "7")).toBe(
      "https://github.com/apps/superplane/installations/new?state=csrf&target_id=7",
    );
  });

  it("puts the personal GitHub account first", () => {
    expect(
      sortGitHubInstallations(
        [
          { id: "11", accountLogin: "acme", accountType: "Organization" },
          { id: "22", accountLogin: "octo", accountType: "User" },
        ],
        "octo",
      ).map((item) => item.accountLogin),
    ).toEqual(["octo", "acme"]);
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
