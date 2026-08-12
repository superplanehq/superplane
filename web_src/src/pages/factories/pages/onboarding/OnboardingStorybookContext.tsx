import { useCallback, useMemo, useState, type ReactNode } from "react";

import type { GitProvider, OnboardingRepo, OnboardingStorybookSeed, PendingOnboarding } from "./onboardingMocks";
import {
  EMPTY_ONBOARDING_NAV_PROGRESS,
  OnboardingStorybookContext,
  type OnboardingNavAnalyzingKey,
  type OnboardingNavProgress,
} from "./onboardingStorybookContextValue";

function nextAnalyzingFlag(wasReady: boolean, nowReady: boolean, current: boolean): boolean {
  if (!nowReady) {
    return false;
  }
  if (!wasReady) {
    return true;
  }
  return current;
}

function analyzingFieldForKey(
  key: OnboardingNavAnalyzingKey,
): keyof Pick<OnboardingNavProgress, "analyzingRepo" | "analyzingIssues" | "analyzingAgent"> {
  if (key === "repo") {
    return "analyzingRepo";
  }
  if (key === "issues") {
    return "analyzingIssues";
  }
  return "analyzingAgent";
}

export function OnboardingStorybookProvider({
  children,
  initial,
}: {
  children: ReactNode;
  initial?: OnboardingStorybookSeed;
}) {
  const [pending, setPending] = useState<PendingOnboarding | null>(initial?.pending ?? null);
  const [connectedProviders, setConnectedProviders] = useState<GitProvider[]>(initial?.connectedProviders ?? []);
  const [enabledReposByWorkspace, setEnabledReposByWorkspace] = useState<Record<string, OnboardingRepo[]>>(
    initial?.enabledReposByWorkspace ?? {},
  );
  const [overviewTipsWorkspaceId, setOverviewTipsWorkspaceId] = useState<string | null>(
    initial?.overviewTipsWorkspaceId ?? null,
  );
  const [setupProgress, setSetupProgress] = useState<OnboardingNavProgress>(EMPTY_ONBOARDING_NAV_PROGRESS);

  const beginOnboarding = useCallback((next: PendingOnboarding) => {
    setPending(next);
    setSetupProgress(EMPTY_ONBOARDING_NAV_PROGRESS);
  }, []);

  const reportSetupProgress = useCallback(
    (progress: Pick<OnboardingNavProgress, "repoReady" | "issuesReady" | "agentReady">) => {
      setSetupProgress((current) => {
        if (
          current.repoReady === progress.repoReady &&
          current.issuesReady === progress.issuesReady &&
          current.agentReady === progress.agentReady
        ) {
          return current;
        }
        return {
          ...current,
          ...progress,
          // Analyze only after Continue commits the step (repoReady / issuesReady here).
          analyzingRepo: nextAnalyzingFlag(current.repoReady, progress.repoReady, current.analyzingRepo),
          analyzingIssues: nextAnalyzingFlag(current.issuesReady, progress.issuesReady, current.analyzingIssues),
          analyzingAgent: nextAnalyzingFlag(current.agentReady, progress.agentReady, current.analyzingAgent),
        };
      });
    },
    [],
  );

  const reportNavAnalyzing = useCallback((key: OnboardingNavAnalyzingKey, analyzing: boolean) => {
    const field = analyzingFieldForKey(key);
    setSetupProgress((current) => (current[field] === analyzing ? current : { ...current, [field]: analyzing }));
  }, []);

  const connectProvider = useCallback((provider: GitProvider) => {
    setConnectedProviders((current) => (current.includes(provider) ? current : [...current, provider]));
  }, []);

  const completeOnboarding = useCallback((workspaceId: string, repos: OnboardingRepo[]) => {
    setEnabledReposByWorkspace((current) => ({ ...current, [workspaceId]: repos }));
    setOverviewTipsWorkspaceId(workspaceId);
    setPending(null);
    setSetupProgress(EMPTY_ONBOARDING_NAV_PROGRESS);
  }, []);

  const clearOverviewTips = useCallback(() => {
    setOverviewTipsWorkspaceId(null);
  }, []);

  const enabledRepos = useCallback(
    (workspaceId: string) => enabledReposByWorkspace[workspaceId] ?? [],
    [enabledReposByWorkspace],
  );

  const shouldShowOverviewTips = useCallback(
    (workspaceId: string) =>
      overviewTipsWorkspaceId === workspaceId && (enabledReposByWorkspace[workspaceId]?.length ?? 0) > 0,
    [enabledReposByWorkspace, overviewTipsWorkspaceId],
  );

  const value = useMemo(
    () => ({
      pending,
      connectedProviders,
      setupProgress,
      reportSetupProgress,
      reportNavAnalyzing,
      beginOnboarding,
      connectProvider,
      completeOnboarding,
      shouldShowOverviewTips,
      clearOverviewTips,
      enabledRepos,
    }),
    [
      pending,
      connectedProviders,
      setupProgress,
      reportSetupProgress,
      reportNavAnalyzing,
      beginOnboarding,
      connectProvider,
      completeOnboarding,
      shouldShowOverviewTips,
      clearOverviewTips,
      enabledRepos,
    ],
  );

  return <OnboardingStorybookContext.Provider value={value}>{children}</OnboardingStorybookContext.Provider>;
}
