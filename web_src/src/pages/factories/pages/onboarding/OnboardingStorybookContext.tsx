import { useCallback, useMemo, useState, type ReactNode } from "react";

import type {
  OnboardingRepo,
  OnboardingStorybookSeed,
  PendingOnboarding,
  WorkspaceConnections,
} from "./onboardingMocks";
import { OnboardingStorybookContext } from "./onboardingStorybookContextValue";

function connectionsFromSeed(initial?: OnboardingStorybookSeed): Record<string, WorkspaceConnections> {
  const connections = { ...(initial?.connectionsByWorkspace ?? {}) };
  for (const [workspaceId, repos] of Object.entries(initial?.enabledReposByWorkspace ?? {})) {
    if (connections[workspaceId]) {
      continue;
    }
    connections[workspaceId] = { repos, issuesChoice: null, issuesRepo: null, agent: null };
  }
  return connections;
}

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
  const [connectionsByWorkspace, setConnectionsByWorkspace] = useState<Record<string, WorkspaceConnections>>(() =>
    connectionsFromSeed(initial),
  );
  const [overviewTipsWorkspaceId, setOverviewTipsWorkspaceId] = useState<string | null>(
    initial?.overviewTipsWorkspaceId ?? null,
  );

  const beginOnboarding = useCallback((next: PendingOnboarding) => {
    setPending(next);
  }, []);

  const completeOnboarding = useCallback(
    (workspaceId: string, repos: OnboardingRepo[], details?: Omit<WorkspaceConnections, "repos">) => {
      setEnabledReposByWorkspace((current) => ({ ...current, [workspaceId]: repos }));
      setConnectionsByWorkspace((current) => ({
        ...current,
        [workspaceId]: {
          repos,
          issuesChoice: details?.issuesChoice ?? null,
          issuesRepo: details?.issuesRepo ?? null,
          agent: details?.agent ?? null,
        },
      }));
      setOverviewTipsWorkspaceId(workspaceId);
      setPending(null);
    },
    [],
  );

  const clearOverviewTips = useCallback(() => {
    setOverviewTipsWorkspaceId(null);
  }, []);

  const enabledRepos = useCallback(
    (workspaceId: string) => enabledReposByWorkspace[workspaceId] ?? [],
    [enabledReposByWorkspace],
  );

  const connections = useCallback(
    (workspaceId: string) => connectionsByWorkspace[workspaceId],
    [connectionsByWorkspace],
  );

  const updateConnections = useCallback((workspaceId: string, next: WorkspaceConnections) => {
    setConnectionsByWorkspace((current) => ({ ...current, [workspaceId]: next }));
    setEnabledReposByWorkspace((current) => ({ ...current, [workspaceId]: next.repos }));
  }, []);

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
      connections,
      updateConnections,
    }),
    [
      pending,
      beginOnboarding,
      completeOnboarding,
      shouldShowOverviewTips,
      clearOverviewTips,
      enabledRepos,
      connections,
      updateConnections,
    ],
  );

  return <OnboardingStorybookContext.Provider value={value}>{children}</OnboardingStorybookContext.Provider>;
}
