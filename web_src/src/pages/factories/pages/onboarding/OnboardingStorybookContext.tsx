import { useCallback, useMemo, useState, type ReactNode } from "react";

import type { GitProvider, OnboardingRepo, OnboardingStorybookSeed, PendingOnboarding } from "./onboardingMocks";
import { OnboardingStorybookContext } from "./onboardingStorybookContextValue";

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

  const beginOnboarding = useCallback((next: PendingOnboarding) => {
    setPending(next);
  }, []);

  const connectProvider = useCallback((provider: GitProvider) => {
    setConnectedProviders((current) => (current.includes(provider) ? current : [...current, provider]));
  }, []);

  const completeOnboarding = useCallback((workspaceId: string, repos: OnboardingRepo[]) => {
    setEnabledReposByWorkspace((current) => ({ ...current, [workspaceId]: repos }));
    setOverviewTipsWorkspaceId(workspaceId);
    setPending(null);
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
