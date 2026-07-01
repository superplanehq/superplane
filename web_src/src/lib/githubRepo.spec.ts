import { describe, expect, it } from "vitest";
import { formatGitHubRepoParam, parseGitHubRepoParam } from "./githubRepo";

describe("parseGitHubRepoParam", () => {
  it("parses github.com owner/repo", () => {
    expect(parseGitHubRepoParam("github.com/superplanehq/preview-env-github-digitalocean")).toEqual({
      owner: "superplanehq",
      repo: "preview-env-github-digitalocean",
    });
  });

  it("parses https github urls", () => {
    expect(parseGitHubRepoParam("https://github.com/acme/widgets.git")).toEqual({
      owner: "acme",
      repo: "widgets",
    });
  });

  it("returns null for invalid values", () => {
    expect(parseGitHubRepoParam("github.com/only-owner")).toBeNull();
  });
});

describe("formatGitHubRepoParam", () => {
  it("formats owner and repo", () => {
    expect(formatGitHubRepoParam("acme", "widgets")).toBe("github.com/acme/widgets");
  });
});
