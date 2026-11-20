import { describe, expect, it } from "bun:test";

import { unconnectedGitHubInstallations } from "./useHostedGitHubConnect";

describe("unconnectedGitHubInstallations", () => {
  it("lists each installed account once and skips connected accounts", () => {
    const repositories = [
      { repositoryId: "1", installationId: "10", accountLogin: "acme" },
      { repositoryId: "2", installationId: "10", accountLogin: "acme" },
      { repositoryId: "3", installationId: "20", accountLogin: "other" },
      { repositoryId: "4", installationId: "30", accountLogin: "linked" },
    ];

    expect(unconnectedGitHubInstallations(repositories, ["github-linked"])).toEqual([
      { installationId: "10", accountLogin: "acme", repositoryId: "1" },
      { installationId: "20", accountLogin: "other", repositoryId: "3" },
    ]);
  });
});
