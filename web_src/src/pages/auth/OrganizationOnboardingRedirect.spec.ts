import { describe, expect, it } from "vitest";

import { getGitHubAccountLinkHref } from "./githubAccountLinkHref";

describe("getGitHubAccountLinkHref", () => {
  it("links GitHub to the current account before resuming onboarding", () => {
    expect(getGitHubAccountLinkHref()).toBe("/auth/github?intent=link&redirect=%2Fonboarding");
  });
});
