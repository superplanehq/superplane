import { createContext } from "react";

/**
 * Re-resolves the provisional workspace behind /onboarding for its current
 * organization slug. Called after the organization is renamed from the GitHub
 * owner, so the retry-safe onboarding endpoint returns the new slug without a
 * full-page reload.
 */
export type OnboardingWorkspaceResolution = () => Promise<void>;

export const OnboardingWorkspaceResolutionContext = createContext<OnboardingWorkspaceResolution | null>(null);
