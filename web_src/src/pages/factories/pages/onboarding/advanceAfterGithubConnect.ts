import type { QueryClient } from "@tanstack/react-query";
import type { NavigateFunction } from "react-router";

import { factoryQueryKeys } from "@/hooks/useFactoryData";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import type { OnboardingWorkspaceResolution } from "./onboardingWorkspaceResolutionContext";
import { onboardingStepPath } from "./onboardingStepPath";

/**
 * Copies the cached factory list and factory detail from the old organization
 * slug to the new one, so the wizard does not fall back to a full-screen
 * loading state while the workspace re-resolves under the new slug.
 */
function seedFactoryQueriesForNewSlug(
  queryClient: QueryClient,
  oldSlug: string,
  nextSlug: string,
  factoryId: string,
): void {
  const list = queryClient.getQueryData(factoryQueryKeys.list(oldSlug));
  if (list !== undefined) {
    queryClient.setQueryData(factoryQueryKeys.list(nextSlug), list);
  }

  const detail = queryClient.getQueryData(factoryQueryKeys.detail(oldSlug, factoryId));
  if (detail !== undefined) {
    queryClient.setQueryData(factoryQueryKeys.detail(nextSlug, factoryId), detail);
  }
}

/** Marks the seeded queries stale, so they refetch in the background under the new slug. */
function invalidateFactoryQueriesForNewSlug(queryClient: QueryClient, nextSlug: string, factoryId: string): void {
  void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.list(nextSlug) });
  void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.detail(nextSlug, factoryId) });
}

/**
 * Moves the wizard from the vcs step to the repo step after GitHub returns.
 *
 * Always uses a client-side navigation. A full reload would flash the GitHub
 * return URL (`step=vcs&pick=newest`) again. Re-resolve the workspace only
 * when the organization slug changed, and do that after the URL already
 * points at the repo step so a remount does not re-select the connection.
 */
export async function advanceAfterGithubConnect(args: {
  onboardingEntryPath?: string | null;
  organizationId: string;
  nextSlug: string;
  factoryId: string;
  factoryKey: string;
  navigate: NavigateFunction;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  queryClient: QueryClient;
}): Promise<void> {
  const nextPath = onboardingStepPath(
    args.onboardingEntryPath ?? factorySetupPath(args.nextSlug, args.factoryKey),
    "repo",
  );
  const slugChanged = Boolean(args.onboardingEntryPath) && args.nextSlug !== args.organizationId;

  if (slugChanged) {
    seedFactoryQueriesForNewSlug(args.queryClient, args.organizationId, args.nextSlug, args.factoryId);
  }

  args.navigate(nextPath, { replace: true });

  if (!slugChanged || !args.reresolveWorkspace) return;

  try {
    await args.reresolveWorkspace();
  } catch {
    // Stay on this document. The repo step is already visible.
  }
  invalidateFactoryQueriesForNewSlug(args.queryClient, args.nextSlug, args.factoryId);
}
