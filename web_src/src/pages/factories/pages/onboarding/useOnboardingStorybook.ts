import { useContext } from "react";

import { OnboardingStorybookContext } from "./onboardingStorybookContextValue";

/** Returns null outside Storybook onboarding harness — app create paths stay unchanged. */
export function useOnboardingStorybook() {
  return useContext(OnboardingStorybookContext);
}
