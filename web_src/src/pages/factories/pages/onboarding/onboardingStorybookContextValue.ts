import { createContext } from "react";

import type { GitProvider, OnboardingRepo, PendingOnboarding } from "./onboardingMocks";

export type OnboardingStorybookContextValue = {
  pending: PendingOnboarding | null;
  connectedProviders: GitProvider[];
  beginOnboarding: (pending: PendingOnboarding) => void;
  connectProvider: (provider: GitProvider) => void;
  completeOnboarding: (workspaceId: string, repos: OnboardingRepo[]) => void;
  shouldShowOverviewTips: (workspaceId: string) => boolean;
  clearOverviewTips: () => void;
  enabledRepos: (workspaceId: string) => OnboardingRepo[];
};

export const OnboardingStorybookContext = createContext<OnboardingStorybookContextValue | null>(null);
