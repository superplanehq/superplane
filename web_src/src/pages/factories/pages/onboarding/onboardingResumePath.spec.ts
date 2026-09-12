import type { FactoriesFactory } from "@/api-client";
import { describe, expect, it } from "vitest";

import { incompleteWorkspaceSetupPath, onboardingResumePath } from "./onboardingResumePath";

describe("onboardingResumePath", () => {
  it("maps first-run onboarding to the workspace setup path", () => {
    expect(onboardingResumePath("acme", "PAY", "")).toBe("/acme/workspaces/pay/setup");
  });

  it("keeps the wizard step and GitHub install-request params", () => {
    expect(
      onboardingResumePath(
        "acme",
        "PAY",
        "?attempt=attempt-1&step=vcs&pick=newest&githubSetup=request&githubOrg=puppies",
      ),
    ).toBe("/acme/workspaces/pay/setup?step=vcs&pick=newest&githubSetup=request&githubOrg=puppies");
  });

  it("drops attempt and other first-run-only params", () => {
    expect(onboardingResumePath("acme", "PAY", "?attempt=attempt-1&auth_error=1")).toBe("/acme/workspaces/pay/setup");
  });
});

describe("incompleteWorkspaceSetupPath", () => {
  it("returns the first incomplete workspace setup path", () => {
    expect(
      incompleteWorkspaceSetupPath("acme", [
        { key: "DONE", onboarding: { completedAt: "2026-09-01T00:00:00.000Z" } } as FactoriesFactory,
        { key: "PAY", onboarding: {} } as FactoriesFactory,
      ]),
    ).toBe("/acme/workspaces/pay/setup");
  });

  it("returns null when every workspace is complete", () => {
    expect(
      incompleteWorkspaceSetupPath("acme", [
        { key: "PAY", onboarding: { completedAt: "2026-09-01T00:00:00.000Z" } } as FactoriesFactory,
      ]),
    ).toBeNull();
  });
});
