import { createContext } from "react";

import type { OnboardingRepo, PendingOnboarding } from "./onboardingMocks";

export type OnboardingStorybookContextValue = {
  pending: PendingOnboarding | null;
  beginOnboarding: (pending: PendingOnboarding) => void;
  completeOnboarding: (workspaceId: string, repos: OnboardingRepo[]) => void;
  shouldShowOverviewTips: (workspaceId: string) => boolean;
  clearOverviewTips: () => void;
  enabledRepos: (workspaceId: string) => OnboardingRepo[];
};

export const OnboardingStorybookContext = createContext<OnboardingStorybookContextValue | null>(null);
