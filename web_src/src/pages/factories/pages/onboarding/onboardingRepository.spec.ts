import { describe, expect, it } from "bun:test";

import { firstRunRepositoryPatch, githubConnectionPatch } from "./onboardingRepository";

describe("firstRunRepositoryPatch", () => {
  it("uses the selected repository for code and GitHub issue intake", () => {
    expect(firstRunRepositoryPatch("github-1", "acme/new")).toEqual({
      vcsIntegrationId: "github-1",
      appRepository: "acme/new",
      backlogRepository: "acme/new",
    });
  });
});

describe("githubConnectionPatch", () => {
  it("clears repository values when the GitHub connection changes", () => {
    expect(githubConnectionPatch("github-2", "github-1")).toEqual({
      vcsIntegrationId: "github-2",
      appRepository: "",
      backlogRepository: "",
      defaultBranch: "",
    });
  });

  it("clears stale repository values when the first GitHub connection is selected", () => {
    expect(githubConnectionPatch("github-1", undefined)).toMatchObject({
      appRepository: "",
      backlogRepository: "",
      defaultBranch: "",
    });
  });

  it("keeps repository values when the GitHub connection does not change", () => {
    expect(githubConnectionPatch("github-1", "github-1")).toEqual({ vcsIntegrationId: "github-1" });
  });
});
