import type { ReactNode } from "react";

import {
  OnboardingWorkspaceResolutionContext,
  type OnboardingWorkspaceResolution,
} from "./onboardingWorkspaceResolutionContext";

export function OnboardingWorkspaceResolutionProvider({
  resolve,
  children,
}: {
  resolve: OnboardingWorkspaceResolution;
  children: ReactNode;
}) {
  return (
    <OnboardingWorkspaceResolutionContext.Provider value={resolve}>
      {children}
    </OnboardingWorkspaceResolutionContext.Provider>
  );
}
