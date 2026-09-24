import { describe, expect, it } from "bun:test";

import { onboardingStepPath } from "./onboardingStepPath";

describe("onboardingStepPath", () => {
  it("keeps the onboarding attempt when GitHub returns", () => {
    expect(onboardingStepPath("/onboarding?attempt=attempt-1", "vcs")).toBe(
      "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
    );
  });

  it("keeps the onboarding route after the organization is named", () => {
    expect(onboardingStepPath("/onboarding?attempt=attempt-1&step=vcs&pick=newest", "repo")).toBe(
      "/onboarding?attempt=attempt-1&step=repo",
    );
  });

  // githubSetup=request and githubOrg are one-shot return flags for the
  // waiting connect screen. Carrying them onto step=repo remounts the wizard
  // on Connect and keeps rechecking a request that is already done.
  it("drops install-request flags when leaving the vcs step", () => {
    expect(
      onboardingStepPath(
        "/onboarding?attempt=attempt-1&githubSetup=request&githubOrg=acme&step=vcs&pick=newest",
        "repo",
      ),
    ).toBe("/onboarding?attempt=attempt-1&step=repo");
  });

  it("uses the organization setup route outside initial onboarding", () => {
    expect(onboardingStepPath("/org-1/workspaces/APP/setup", "agent")).toBe("/org-1/workspaces/APP/setup?step=agent");
  });
});
