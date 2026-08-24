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
  const beginOnboarding = useCallback((next: PendingOnboarding) => {
    setPending(next);
  }, []);

  const completeOnboarding = useCallback((workspaceId: string, repos: OnboardingRepo[]) => {
    setEnabledReposByWorkspace((current) => ({ ...current, [workspaceId]: repos }));
    setPending(null);
  }, []);

  const enabledRepos = useCallback(
    (workspaceId: string) => enabledReposByWorkspace[workspaceId] ?? [],
    [enabledReposByWorkspace],
  );

  const value = useMemo(
    () => ({
      pending,
      beginOnboarding,
      completeOnboarding,
      enabledRepos,
    }),
    [pending, beginOnboarding, completeOnboarding, enabledRepos],
  );

  return <OnboardingStorybookContext.Provider value={value}>{children}</OnboardingStorybookContext.Provider>;
}
