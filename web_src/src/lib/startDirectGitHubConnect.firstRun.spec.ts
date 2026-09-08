import { describe, expect, it } from "vitest";

import { firstRunGithubOrganization, pendingGitHubRequestedPicker } from "./startDirectGitHubConnect";

describe("pendingGitHubRequestedPicker", () => {
  it("uses the request that has state when an earlier request only has an account", () => {
    expect(
      pendingGitHubRequestedPicker(
        [
          {
            metadata: { id: "int-stale", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                installRequested: true,
                installRequestedAccount: "acme",
              },
            },
          },
          {
            metadata: { id: "int-live", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                installRequested: true,
                installRequestedAccount: "globex",
                state: "csrf",
              },
            },
          },
        ],
        "user-1",
        "acme",
      ),
    ).toEqual({
      id: "int-live",
      state: "csrf",
      appSlug: "",
      authorizeUrl: "",
      githubLogin: "",
      requestedAccount: "globex",
      installations: [],
    });
  });
});

describe("firstRunGithubOrganization", () => {
  it("uses the picker request when an earlier request has an account and no state", () => {
    expect(
      firstRunGithubOrganization(
        "acme",
        {
          id: "int-live",
          installations: [],
          state: "csrf",
          appSlug: "",
          authorizeUrl: "",
          githubLogin: "",
          requestedAccount: "globex",
        },
        [
          {
            status: { metadata: { installRequestedAccount: "acme" } },
          },
        ],
      ),
    ).toBe("globex");
  });

  it("uses the return query when no picker is ready", () => {
    expect(firstRunGithubOrganization("acme", undefined, [])).toBe("acme");
  });
});
