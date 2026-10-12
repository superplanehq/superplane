import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { useEffect, useState } from "react";
import { useLocation } from "react-router";

import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import { navigateGitHubWindow, openGitHubWindow } from "./useFirstRunGitHub";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import { useBitbucketOnboarding } from "./useBitbucketOnboarding";
import { onboardingStepPath } from "./onboardingStepPath";

type InstallationAttempt = {
  startedAt: number;
  knownSlugs: string[];
};

function installedSlugs(data: { installedWorkspaces?: Array<{ slug?: string; externalId?: string }> } | undefined) {
  return (data?.installedWorkspaces ?? [])
    .map((workspace) => workspace.slug || workspace.externalId || "")
    .filter(Boolean);
}

export function useFirstRunBitbucket(args: {
  organizationId: string;
  setupFinished: boolean;
  blocking: FirstRunBlocking;
}) {
  const location = useLocation();
  const [attempt, setAttempt] = useState<InstallationAttempt | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const timedOut = attempt != null && now - attempt.startedAt >= bitbucketInstallationAttemptMs;
  const attemptActive = attempt != null && !timedOut;
  const onboarding = useBitbucketOnboarding(args.organizationId, {
    poll: !args.setupFinished,
    attemptActive,
  });
  const slugs = installedSlugs(onboarding.data);
  const refetchOnboarding = onboarding.refetch;

  // Stop the attempt when a new installation appears. Existing selections
  // stay untouched, and the picker shows the new workspace automatically.
  useEffect(() => {
    if (attempt == null) return;
    if (slugs.some((slug) => !attempt.knownSlugs.includes(slug))) setAttempt(null);
  }, [attempt, slugs.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps

  // Re-render once when the five-minute window ends.
  useEffect(() => {
    if (attempt == null || timedOut) return;
    const remaining = bitbucketInstallationAttemptMs - (Date.now() - attempt.startedAt);
    const timer = setTimeout(() => setNow(Date.now()), remaining);
    return () => clearTimeout(timer);
  }, [attempt, timedOut]);

  // Refetch immediately when the user returns from Bitbucket.
  useEffect(() => {
    if (!attemptActive) return;
    const refetch = () => void refetchOnboarding();
    const onVisibility = () => {
      if (!document.hidden) refetch();
    };
    window.addEventListener("focus", refetch);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      window.removeEventListener("focus", refetch);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [attemptActive, refetchOnboarding]);

  const startAttempt = (known: string[]) => setAttempt({ startedAt: Date.now(), knownSlugs: known });

  const grantAccess = () =>
    args.blocking.run("opening-bitbucket", async () => {
      const popup = openGitHubWindow();
      const known = installedSlugs(onboarding.data);
      try {
        const url = await onboarding.startInstallation.mutateAsync();
        startAttempt(known);
        navigateGitHubWindow(popup, url);
      } catch (error) {
        popup?.close();
        throw error;
      }
    });
  return {
    configured: Boolean(onboarding.data?.providerConfigured),
    pending: onboarding.isPending && !onboarding.data,
    repositories: (onboarding.data?.repositories ?? []).map((repository) => repository.fullName ?? "").filter(Boolean),
    installedWorkspaces: slugs,
    installationAttemptActive: attemptActive,
    installationAttemptTimedOut: timedOut,
    identityLinked: Boolean(onboarding.data?.identity?.login || onboarding.data?.identity?.providerUserId),
    loadError: Boolean(onboarding.error),
    lookupFailed: Boolean(onboarding.error) && !onboarding.data,
    lookupRetrying: Boolean(onboarding.isFetching),
    retryLookup: () => {
      // Check again refetches and restarts the five-minute watch.
      if (timedOut) startAttempt(installedSlugs(onboarding.data));
      return void onboarding.refetch();
    },
    connectHref: linkedAccountConnectHref(
      "bitbucket",
      onboardingStepPath(`${location.pathname}${location.search}`, "repo"),
    ),
    grantAccess,
  };
}
