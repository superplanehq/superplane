import { describe, expect, it } from "bun:test";

import { closerFromGitHubMergedBy, githubPullRequestApiHref } from "./githubPullRequestMerger";

describe("githubPullRequestApiHref", () => {
  it("maps a GitHub pull request page to the GitHub API", () => {
    expect(githubPullRequestApiHref("https://github.com/superplanehq/superplane/pull/8044")).toBe(
      "https://api.github.com/repos/superplanehq/superplane/pulls/8044",
    );
  });

  it("maps a GitHub Enterprise Cloud pull request page to the tenant API", () => {
    expect(githubPullRequestApiHref("https://acme.ghe.com/superplanehq/superplane/pull/8044")).toBe(
      "https://api.acme.ghe.com/repos/superplanehq/superplane/pulls/8044",
    );
  });

  it("ignores non-GitHub hosts", () => {
    expect(githubPullRequestApiHref("https://gitlab.com/acme/app/pull/12")).toBeUndefined();
    expect(githubPullRequestApiHref("https://ghes.corp.example/acme/app/pull/12")).toBeUndefined();
  });
});

describe("closerFromGitHubMergedBy", () => {
  it("names the merger and links the GitHub profile", () => {
    expect(
      closerFromGitHubMergedBy({
        login: "alex",
        name: "Alex Rivera",
        html_url: "https://github.com/alex",
      }),
    ).toEqual({ automationName: "Alex Rivera", automationHref: "https://github.com/alex" });
  });

  it("uses the login when GitHub has no display name", () => {
    expect(closerFromGitHubMergedBy({ login: "alex" })).toEqual({
      automationName: "alex",
      automationHref: "https://github.com/alex",
    });
  });

  it("builds the profile URL on the tenant host when GitHub omits html_url", () => {
    expect(closerFromGitHubMergedBy({ login: "alex" }, "acme.ghe.com")).toEqual({
      automationName: "alex",
      automationHref: "https://acme.ghe.com/alex",
    });
  });
});
