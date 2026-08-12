import { useCallback, useMemo, useState, type ReactNode } from "react";

import type { OnboardingRepo, OnboardingStorybookSeed, PendingOnboarding } from "./onboardingMocks";
import { OnboardingStorybookContext } from "./onboardingStorybookContextValue";

export function OnboardingStorybookProvider({
  children,
  initial,
}: {
  children: ReactNode;
  initial?: OnboardingStorybookSeed;
}) {
  const [pending, setPending] = useState<PendingOnboarding | null>(initial?.pending ?? null);
  const [enabledReposByWorkspace, setEnabledReposByWorkspace] = useState<Record<string, OnboardingRepo[]>>(
    initial?.enabledReposByWorkspace ?? {},
  );
  const [overviewTipsWorkspaceId, setOverviewTipsWorkspaceId] = useState<string | null>(
    initial?.overviewTipsWorkspaceId ?? null,
  );

  const beginOnboarding = useCallback((next: PendingOnboarding) => {
    setPending(next);
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
      beginOnboarding,
      completeOnboarding,
      shouldShowOverviewTips,
      clearOverviewTips,
      enabledRepos,
    }),
    [pending, beginOnboarding, completeOnboarding, shouldShowOverviewTips, clearOverviewTips, enabledRepos],
  );

  return <OnboardingStorybookContext.Provider value={value}>{children}</OnboardingStorybookContext.Provider>;
}
