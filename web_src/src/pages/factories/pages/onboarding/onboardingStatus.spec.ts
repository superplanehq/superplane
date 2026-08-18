import { describe, expect, it } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { isFactoryOnboardingComplete } from "./onboardingStatus";

describe("isFactoryOnboardingComplete", () => {
  it("returns false while completion time is absent", () => {
    expect(isFactoryOnboardingComplete({ onboarding: {} } as FactoriesFactory)).toBe(false);
  });

  it("returns true when onboarding is complete", () => {
    expect(
      isFactoryOnboardingComplete({
        onboarding: { completedAt: "2026-08-17T12:00:00Z" },
      } as FactoriesFactory),
    ).toBe(true);
  });
});
