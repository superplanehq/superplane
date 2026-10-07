import { describe, expect, it } from "bun:test";

import { onboardingStepPath } from "./onboardingStepPath";

describe("onboardingStepPath", () => {
  it("keeps the onboarding attempt", () => {
    expect(onboardingStepPath("/onboarding?attempt=attempt-1", "vcs")).toBe("/onboarding?attempt=attempt-1&step=vcs");
  });

  it("keeps the onboarding route when the step changes", () => {
    expect(onboardingStepPath("/onboarding?attempt=attempt-1&step=vcs", "repo")).toBe(
      "/onboarding?attempt=attempt-1&step=repo",
    );
  });

  it("drops parameters from the removed hosted integration flow", () => {
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
