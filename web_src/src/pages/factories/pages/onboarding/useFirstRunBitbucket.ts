import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { useLocation } from "react-router";

import { navigateGitHubWindow, openGitHubWindow } from "./useFirstRunGitHub";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import { useBitbucketOnboarding } from "./useBitbucketOnboarding";
import { onboardingStepPath } from "./onboardingStepPath";

export function useFirstRunBitbucket(args: {
  organizationId: string;
  setupFinished: boolean;
  blocking: FirstRunBlocking;
}) {
  const location = useLocation();
  const onboarding = useBitbucketOnboarding(args.organizationId, { poll: !args.setupFinished });
  const grantAccess = () =>
    args.blocking.run("opening-bitbucket", async () => {
      const popup = openGitHubWindow();
      try {
        const url = await onboarding.startInstallation.mutateAsync();
        navigateGitHubWindow(popup, url);
      } catch (error) {
        popup?.close();
        throw error;
      }
    });
  return {
    configured: Boolean(onboarding.data?.providerConfigured),
    pending: onboarding.isPending && !onboarding.data,
    installUrl: onboarding.data?.installUrl ?? "",
    repositories: (onboarding.data?.repositories ?? []).map((repository) => repository.fullName ?? "").filter(Boolean),
    installedWorkspaces: (onboarding.data?.installedWorkspaces ?? [])
      .map((workspace) => workspace.slug || workspace.externalId || "")
      .filter(Boolean),
    identityLinked: Boolean(onboarding.data?.identity?.login || onboarding.data?.identity?.providerUserId),
    loadError: Boolean(onboarding.error),
    lookupFailed: Boolean(onboarding.error) && !onboarding.data,
    lookupRetrying: Boolean(onboarding.isFetching),
    retryLookup: () => void onboarding.refetch(),
    connectHref: linkedAccountConnectHref(
      "bitbucket",
      onboardingStepPath(`${location.pathname}${location.search}`, "repo"),
    ),
    grantAccess,
  };
}
