import { useState } from "react";
import { useLocation } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { factoriesConnectVcsProviderInstallation, type MeVcsProviderRepository } from "@/api-client";
import { integrationKeys } from "@/hooks/useIntegrations";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { navigateGitHubWindow, openGitHubWindow } from "@/pages/factories/pages/onboarding/useFirstRunGitHub";
import { useGitHubOnboarding } from "@/pages/factories/pages/onboarding/useGitHubOnboarding";

type GitHubInstallation = { installationId: string; accountLogin: string; repositoryId: string };

/** One entry per installed GitHub account that has no organization connection yet. */
export function unconnectedGitHubInstallations(
  repositories: MeVcsProviderRepository[] | undefined,
  connectedNames: string[],
): GitHubInstallation[] {
  const installations = new Map<string, GitHubInstallation>();
  for (const repository of repositories ?? []) {
    const { installationId, accountLogin, repositoryId } = repository;
    if (!installationId || !accountLogin || !repositoryId || installations.has(installationId)) continue;
    if (connectedNames.includes(`github-${accountLogin}`)) continue;
    installations.set(installationId, { installationId, accountLogin, repositoryId });
  }
  return [...installations.values()];
}

/**
 * Public GitHub App connect for the workspace integrations page. It uses the
 * same steps as workspace onboarding: link the GitHub account, install the
 * App, then connect each installed account to the organization.
 */
export function useHostedGitHubConnect(organizationId: string, connectedNames: string[]) {
  // Poll only after the user opens GitHub. The setup callback syncs the new
  // installation in the background, so one refetch on return can miss it.
  const [waiting, setWaiting] = useState(false);
  const github = useGitHubOnboarding(organizationId, { poll: waiting });
  const location = useLocation();
  const queryClient = useQueryClient();
  const [opening, setOpening] = useState(false);
  const connectInstallation = useMutation({
    mutationFn: async (installation: GitHubInstallation) => {
      await factoriesConnectVcsProviderInstallation(
        withOrganizationHeader({ organizationId, body: { repositoryId: installation.repositoryId } }),
      );
      return installation;
    },
    onSuccess: (installation) => {
      showSuccessToast(`Connected ${installation.accountLogin}.`);
      setWaiting(false);
      void queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
      void github.refetch();
    },
    onError: (error) => showErrorToast(getApiErrorMessage(error, "SuperPlane could not connect GitHub. Try again.")),
  });

  const start = async () => {
    if (!github.data?.identity) {
      window.location.assign(linkedAccountConnectHref("github", `${location.pathname}${location.search}`));
      return;
    }
    setOpening(true);
    const popup = openGitHubWindow();
    try {
      navigateGitHubWindow(popup, await github.startInstallation.mutateAsync());
      setWaiting(true);
    } catch (error) {
      popup?.close();
      showErrorToast(getApiErrorMessage(error, "SuperPlane could not open GitHub. Try again."));
    } finally {
      setOpening(false);
    }
  };

  return {
    loading: github.isPending,
    enabled: github.data?.accountConnectionRequired === true,
    opening,
    start,
    installations: unconnectedGitHubInstallations(github.data?.repositories, connectedNames),
    connectInstallation,
  };
}
