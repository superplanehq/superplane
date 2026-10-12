import { useAccount } from "@/contexts/useAccount";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useRef } from "react";

import { githubAccessKeys, useGitHubInstallReturn } from "./githubInstallReturn";
import { useOnboardingGitHubConnect } from "./onboardingGitHubConnect";
import { useGitHubOnboarding } from "./useGitHubOnboarding";

const ORGANIZATION_NOT_FOUND = "Not Found";

export function githubOnboardingMessage(error: unknown, fallback: string): string {
  const message = getApiErrorMessage(error, fallback);
  if (message === ORGANIZATION_NOT_FOUND) return fallback;
  return message;
}

export function openGitHubWindow(): Window | null {
  const popup = window.open("about:blank", "_blank");
  if (popup) popup.opener = null;
  return popup;
}

export function navigateGitHubWindow(popup: Window | null, url: string) {
  if (popup) {
    popup.location.replace(url);
    return;
  }
  window.location.assign(url);
}

function useGitHubConnectionState(organizationId: string, options?: { poll?: boolean }) {
  const onboarding = useGitHubOnboarding(organizationId, options);
  return {
    onboarding,
    identity: onboarding.data?.identity,
    identities: onboarding.data?.identities ?? [],
    repositories: onboarding.data?.repositories ?? [],
    // An empty login still means a pending request; the screen shows generic copy for it.
    pendingOrganizations: [
      ...new Set((onboarding.data?.pendingRequests ?? []).map((request) => request.accountLogin?.trim() ?? "")),
    ],
    synchronizing: Boolean(onboarding.data?.synchronizing),
    appConfigured: Boolean(onboarding.data?.providerConfigured),
    accountConnectionRequired: onboarding.data?.accountConnectionRequired === true,
  };
}

export type GitHubConnectionState = ReturnType<typeof useGitHubConnectionState>;

function useRepositoryErrorToast(error: unknown, reportErrors: boolean) {
  const reported = useRef<unknown>(null);
  useEffect(() => {
    if (!reportErrors || !error || reported.current === error) return;
    reported.current = error;
    showErrorToast(githubOnboardingMessage(error, "Failed to load repositories"));
  }, [error, reportErrors]);
}

/**
 * GitHub access for this person in this workspace. A GitHub identity is ready
 * only after the person connects GitHub here.
 */
export function useFirstRunGitHub(args: {
  organizationId: string;
  factoryId: string;
  setupFinished: boolean;
  connectedBefore: boolean;
}) {
  const { organizationId, factoryId, setupFinished, connectedBefore } = args;
  const connection = useGitHubConnectionState(organizationId, setupFinished ? { poll: false } : undefined);
  useRepositoryErrorToast(connection.onboarding.error, !setupFinished);
  const { account } = useAccount();
  const accountId = account?.id ?? "";
  const githubConnected = useOnboardingGitHubConnect({ accountId, factoryId, connectedBefore });
  const installScope = { accountId, factoryId };
  const checkingGitHub = useGitHubInstallReturn(
    installScope,
    githubAccessKeys(connection.onboarding.data),
    connection.onboarding.refetch,
  );
  return {
    connection,
    githubReady: connection.accountConnectionRequired
      ? Boolean(connection.identity) && githubConnected
      : connection.appConfigured,
    installScope,
    checkingGitHub,
  };
}
