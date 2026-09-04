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

  it("strips .git before trailing slash", () => {
    expect(parseGitHubRepoParam("https://github.com/acme/widgets.git/")).toEqual({
      owner: "acme",
      repo: "widgets",
    });
  });

  it("strips .git before multiple trailing slashes", () => {
    expect(parseGitHubRepoParam("https://github.com/acme/widgets.git///")).toEqual({
      owner: "acme",
      repo: "widgets",
    });
  });

  it("strips trailing slash with no scheme", () => {
    expect(parseGitHubRepoParam("github.com/acme/widgets/")).toEqual({
      owner: "acme",
      repo: "widgets",
    });
  });
});

describe("formatGitHubRepoParam", () => {
  it("formats owner and repo", () => {
    expect(formatGitHubRepoParam("acme", "widgets")).toBe("github.com/acme/widgets");
  });
});
