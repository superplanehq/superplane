import { useContext } from "react";

import {
  OnboardingWorkspaceResolutionContext,
  type OnboardingWorkspaceResolution,
} from "./onboardingWorkspaceResolutionContext";

export function useOnboardingWorkspaceResolution(): OnboardingWorkspaceResolution | null {
  return useContext(OnboardingWorkspaceResolutionContext);
}
