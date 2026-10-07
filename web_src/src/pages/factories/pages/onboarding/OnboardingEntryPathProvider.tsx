import type { ReactNode } from "react";

import { OnboardingEntryPathContext } from "./onboardingEntryPathContext";

export function OnboardingEntryPathProvider({ path, children }: { path: string; children: ReactNode }) {
  return <OnboardingEntryPathContext.Provider value={path}>{children}</OnboardingEntryPathContext.Provider>;
}
