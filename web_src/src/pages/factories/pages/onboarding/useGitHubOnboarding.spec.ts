import { describe, expect, it } from "bun:test";

import { githubOnboardingPollInterval } from "./useGitHubOnboarding";

describe("githubOnboardingPollInterval", () => {
  it("does not poll before a GitHub identity is connected", () => {
    expect(githubOnboardingPollInterval(undefined)).toBe(false);
    expect(githubOnboardingPollInterval({ synchronizing: true })).toBe(false);
  });

  it("polls organization selection when sign-in is not required", () => {
    expect(
      githubOnboardingPollInterval({
        providerConfigured: true,
        accountConnectionRequired: false,
        synchronizing: false,
      }),
    ).toBe(3_000);
  });

  it("polls every three seconds while waiting for GitHub changes", () => {
    expect(
      githubOnboardingPollInterval({
        identity: { userId: "9", login: "octocat" },
        synchronizing: false,
      }),
    ).toBe(3_000);
  });

  it("polls every second while repositories synchronize", () => {
    expect(
      githubOnboardingPollInterval({
        identity: { userId: "9", login: "octocat" },
        synchronizing: true,
      }),
    ).toBe(1_000);
  });
});
