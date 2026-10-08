import { describe, expect, it } from "bun:test";

import { bitbucketOnboardingPollInterval } from "./bitbucketOnboardingPoll";

const linked = {
  providerConfigured: true,
  identity: { login: "ada" },
  repositories: [],
};

describe("bitbucketOnboardingPollInterval", () => {
  it("does not poll when onboarding polling is off", () => {
    expect(bitbucketOnboardingPollInterval(false, { status: "error" })).toBe(false);
    expect(bitbucketOnboardingPollInterval(false, { status: "success", data: linked })).toBe(false);
  });

  it("keeps polling after a failed workspace check", () => {
    expect(bitbucketOnboardingPollInterval(true, { status: "error" })).toBe(3_000);
  });

  it("polls while a linked account is waiting for repositories", () => {
    expect(bitbucketOnboardingPollInterval(true, { status: "success", data: linked })).toBe(3_000);
  });

  it("stops polling before an account is linked or after repositories appear", () => {
    expect(bitbucketOnboardingPollInterval(true, { status: "success" })).toBe(false);
    expect(
      bitbucketOnboardingPollInterval(true, {
        status: "success",
        data: { providerConfigured: true, repositories: [{ fullName: "acme/api" }] },
      }),
    ).toBe(false);
    expect(
      bitbucketOnboardingPollInterval(true, {
        status: "success",
        data: { ...linked, repositories: [{ fullName: "acme/api" }] },
      }),
    ).toBe(false);
  });
});
