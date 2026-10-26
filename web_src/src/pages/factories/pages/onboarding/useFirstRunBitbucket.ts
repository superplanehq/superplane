import type { MeDescribeVcsProviderOnboardingResponse } from "@/api-client";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { useEffect, useState } from "react";
import { useLocation } from "react-router";

import {
  clearBitbucketInstallationAttempt,
  readBitbucketInstallationAttempt,
  writeBitbucketInstallationAttempt,
  type BitbucketInstallationAttempt,
} from "./bitbucketInstallationAttempt";
import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import { navigateGitHubWindow, openGitHubWindow } from "./useFirstRunGitHub";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import { useBitbucketOnboarding } from "./useBitbucketOnboarding";
import { onboardingStepPath } from "./onboardingStepPath";

type OrganizationAttempt = {
  organizationId: string;
  attempt: BitbucketInstallationAttempt | null;
};

// Restore the attempt that a reload interrupted. Drop one that already expired.
function restoreAttempt(organizationId: string): OrganizationAttempt {
  const stored = readBitbucketInstallationAttempt(organizationId);
  if (stored && Date.now() - stored.startedAt >= bitbucketInstallationAttemptMs) {
    clearBitbucketInstallationAttempt(organizationId);
    return { organizationId, attempt: null };
  }
  return { organizationId, attempt: stored };
}

function installedSlugs(data: { installedWorkspaces?: Array<{ slug?: string; externalId?: string }> } | undefined) {
  return (data?.installedWorkspaces ?? [])
    .map((workspace) => workspace.slug || workspace.externalId || "")
    .filter(Boolean);
}

function linkedIdentities(data: Pick<MeDescribeVcsProviderOnboardingResponse, "identity" | "identities"> | undefined) {
  const login = data?.identity?.login ?? "";
  const providerUserId = data?.identity?.providerUserId ?? "";
  return {
    identityLinked: Boolean(login || providerUserId),
    login,
    providerUserId,
    identities: data?.identities ?? [],
  };
}

export function useFirstRunBitbucket(args: {
  organizationId: string;
  setupFinished: boolean;
  blocking: FirstRunBlocking;
}) {
  const location = useLocation();
  const { organizationId } = args;
  const [stored, setStored] = useState(() => restoreAttempt(organizationId));
  // An organization change must never show the spinner of another organization.
  let current = stored;
  if (stored.organizationId !== organizationId) {
    current = restoreAttempt(organizationId);
    setStored(current);
  }
  const attempt = current.attempt;
  const [now, setNow] = useState(() => Date.now());
  const timedOut = attempt != null && now - attempt.startedAt >= bitbucketInstallationAttemptMs;
  const attemptActive = attempt != null && !timedOut;
  const onboarding = useBitbucketOnboarding(organizationId, {
    poll: !args.setupFinished,
    attemptActive,
  });
  const slugs = installedSlugs(onboarding.data);
  const refetchOnboarding = onboarding.refetch;

  const startAttempt = (known: string[]) => {
    const next = { startedAt: Date.now(), knownSlugs: known };
    writeBitbucketInstallationAttempt(organizationId, next);
    setStored({ organizationId, attempt: next });
  };

  const endAttempt = () => {
    clearBitbucketInstallationAttempt(organizationId);
    setStored({ organizationId, attempt: null });
  };

  // Stop the attempt when a new installation appears. Existing selections
  // stay untouched, and the picker shows the new workspace automatically.
  useEffect(() => {
    if (attempt == null) return;
    if (slugs.some((slug) => !attempt.knownSlugs.includes(slug))) endAttempt();
  }, [attempt, slugs.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps

  // Forget the stored attempt when the five-minute window ends. The state
  // keeps it, so the timed-out notice and the retry path still work.
  useEffect(() => {
    if (timedOut) clearBitbucketInstallationAttempt(organizationId);
  }, [timedOut, organizationId]);

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

  const installUrl = onboarding.data?.installUrl ?? "";

  const copyInstallLink = async () => {
    const url = onboarding.data?.installUrl ?? (await onboarding.startInstallation.mutateAsync());
    await navigator.clipboard.writeText(url);
    startAttempt(installedSlugs(onboarding.data));
  };

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
    ...linkedIdentities(onboarding.data),
    selectIdentity: (providerUserId: string) => onboarding.selectIdentity.mutateAsync(providerUserId),
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
    installUrl,
    copyInstallLink,
    grantAccess,
  };
}
