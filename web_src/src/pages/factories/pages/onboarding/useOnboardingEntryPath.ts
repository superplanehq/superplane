import { useContext } from "react";

import { OnboardingEntryPathContext } from "./onboardingEntryPathContext";

export function useOnboardingEntryPath(): string | null {
  return useContext(OnboardingEntryPathContext);
}
