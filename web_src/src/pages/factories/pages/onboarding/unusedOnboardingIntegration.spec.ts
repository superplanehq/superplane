import { describe, expect, it } from "vitest";

import { unusedOnboardingVcsIntegrationId } from "./unusedOnboardingIntegration";

describe("unusedOnboardingVcsIntegrationId", () => {
  it("returns the previous id on initial setup when nothing else uses it", () => {
    expect(
      unusedOnboardingVcsIntegrationId({
        isInitial: true,
        previousId: "old",
        nextId: "new",
        currentFactoryId: "factory-1",
        factories: [{ id: "factory-1", onboarding: { vcsIntegrationId: "old" } }],
      }),
    ).toBe("old");
  });

  it("keeps the previous id when another workspace still uses it", () => {
    expect(
      unusedOnboardingVcsIntegrationId({
        isInitial: true,
        previousId: "old",
        nextId: "new",
        currentFactoryId: "factory-1",
        factories: [
          { id: "factory-1", onboarding: { vcsIntegrationId: "old" } },
          { id: "factory-2", onboarding: { vcsIntegrationId: "old" } },
        ],
      }),
    ).toBeUndefined();
  });

  it("does not delete during workspace onboarding", () => {
    expect(
      unusedOnboardingVcsIntegrationId({
        isInitial: false,
        previousId: "old",
        nextId: "new",
        currentFactoryId: "factory-1",
        factories: [{ id: "factory-1", onboarding: { vcsIntegrationId: "old" } }],
      }),
    ).toBeUndefined();
  });

  it("does not delete when the selection did not change", () => {
    expect(
      unusedOnboardingVcsIntegrationId({
        isInitial: true,
        previousId: "same",
        nextId: "same",
        currentFactoryId: "factory-1",
        factories: [{ id: "factory-1", onboarding: { vcsIntegrationId: "same" } }],
      }),
    ).toBeUndefined();
  });
});
