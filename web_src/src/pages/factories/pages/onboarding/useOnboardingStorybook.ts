import { useContext } from "react";

import { OnboardingStorybookContext } from "./onboardingStorybookContextValue";

/** Returns null outside the Storybook onboarding harness. */
export function useOnboardingStorybook() {
  return useContext(OnboardingStorybookContext);
}
