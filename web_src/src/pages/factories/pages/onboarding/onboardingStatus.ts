import type { FactoriesFactory } from "@/api-client";

export function isFactoryOnboardingComplete(factory: FactoriesFactory | null | undefined): boolean {
  return Boolean(factory?.onboarding?.completedAt);
}
