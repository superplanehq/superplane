import { describe, expect, it } from "vitest";

import {
  hostedGitHubAppSlug,
  hostedGitHubBindPath,
  hostedGitHubInstallURL,
  hostedGitHubState,
  pendingGitHubInstallations,
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
  });

  it("reads state and slug", () => {
    expect(hostedGitHubState({ state: "csrf" })).toBe("csrf");
    expect(hostedGitHubAppSlug({ githubApp: { slug: "superplane" } })).toBe("superplane");
  });
});
