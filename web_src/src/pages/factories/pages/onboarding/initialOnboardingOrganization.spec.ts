import { describe, expect, it } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { shouldNameOrganizationFromGitHub } from "./initialOnboardingOrganization";

function factoryWithOnboarding(onboarding: FactoriesFactory["onboarding"]): FactoriesFactory {
  return { onboarding } as FactoriesFactory;
}

describe("shouldNameOrganizationFromGitHub", () => {
  it("names only an organization from the initial account onboarding workspace", () => {
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({ initial: true }), true)).toBe(true);
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({}), true)).toBe(false);
  });

  it("does not rename after the onboarding flow saves a GitHub integration", () => {
    const factory = factoryWithOnboarding({ initial: true, vcsIntegrationId: "integration-id" });
    expect(shouldNameOrganizationFromGitHub(factory, true)).toBe(false);
  });

  it("does not rename when the user selects an existing connection", () => {
    expect(shouldNameOrganizationFromGitHub(factoryWithOnboarding({ initial: true }), false)).toBe(false);
  });
});
