import { describe, expect, it } from "vitest";
import { buildGitRef, parseGitRef } from "./gitRef";

describe("parseGitRef", () => {
  it("parses branch refs", () => {
    expect(parseGitRef("refs/heads/main")).toEqual({ kind: "branch", name: "main" });
  });

  it("parses tag refs", () => {
    expect(parseGitRef("refs/tags/v1.0.0")).toEqual({ kind: "tag", name: "v1.0.0" });
  });

  it("parses pull request refs", () => {
    expect(parseGitRef("refs/pull/42")).toEqual({ kind: "pull-request", name: "42" });
  });

  it("normalizes GitHub-style pull request suffixes to the PR number", () => {
    expect(parseGitRef("refs/pull/42/merge")).toEqual({ kind: "pull-request", name: "42" });
    expect(parseGitRef("refs/pull/99/head")).toEqual({ kind: "pull-request", name: "99" });
  });
});

describe("buildGitRef", () => {
  it("builds branch refs", () => {
    expect(buildGitRef("branch", "main")).toBe("refs/heads/main");
  });

  it("builds tag refs", () => {
    expect(buildGitRef("tag", "v1.0.0")).toBe("refs/tags/v1.0.0");
  });

  it("builds pull request refs", () => {
    expect(buildGitRef("pull-request", "123")).toBe("refs/pull/123");
  });

  it("strips GitHub-style pull request suffixes when building", () => {
    expect(buildGitRef("pull-request", "123/merge")).toBe("refs/pull/123");
    expect(buildGitRef("pull-request", "123/head")).toBe("refs/pull/123");
  });
});
