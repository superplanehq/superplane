import { createContext } from "react";

import type { OnboardingRepo, PendingOnboarding, WorkspaceConnections } from "./onboardingMocks";

export type OnboardingStorybookContextValue = {
  pending: PendingOnboarding | null;
  beginOnboarding: (pending: PendingOnboarding) => void;
  completeOnboarding: (
    workspaceId: string,
    repos: OnboardingRepo[],
    details?: Omit<WorkspaceConnections, "repos">,
  ) => void;
  shouldShowOverviewTips: (workspaceId: string) => boolean;
  clearOverviewTips: () => void;
  enabledRepos: (workspaceId: string) => OnboardingRepo[];
  connections: (workspaceId: string) => WorkspaceConnections | undefined;
  updateConnections: (workspaceId: string, next: WorkspaceConnections) => void;
};

export const OnboardingStorybookContext = createContext<OnboardingStorybookContextValue | null>(null);
