import {
  meConfigureVcsProviderInstallation,
  meDescribeVcsProviderOnboarding,
  meRefreshVcsProviderOnboarding,
  meSelectVcsProviderOnboardingIdentity,
  meStartVcsProviderInstallation,
  meVerifyVcsProviderInstallations,
  type MeDescribeVcsProviderOnboardingResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const githubOnboardingKey = (organizationId: string) => ["me", organizationId, "github-onboarding"] as const;
const githubInstallationChecksKey = (organizationId: string) =>
  ["me", organizationId, "github-installation-checks"] as const;
const githubProvider = "github";
const githubOnboardingPollIntervalMs = 3_000;
const githubOnboardingSyncPollIntervalMs = 1_000;
const githubInstallationCheckIntervalMs = 10_000;

type GitHubOnboardingPollingState = Pick<
  MeDescribeVcsProviderOnboardingResponse,
  "identity" | "synchronizing" | "providerConfigured" | "accountConnectionRequired"
>;

export function githubOnboardingPollInterval(data: GitHubOnboardingPollingState | undefined): number | false {
  if (!data) return false;
  const appOnly = data.providerConfigured === true && data.accountConnectionRequired !== true;
  if (!data.identity && !appOnly) return false;
  return data.synchronizing ? githubOnboardingSyncPollIntervalMs : githubOnboardingPollIntervalMs;
}

export function useGitHubOnboarding(organizationId: string, options: { poll?: boolean } = {}) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: githubOnboardingKey(organizationId),
    queryFn: async () => {
      const response = await meDescribeVcsProviderOnboarding(
        withOrganizationHeader({ organizationId, path: { provider: githubProvider } }),
      );
      return response.data ?? ({} as MeDescribeVcsProviderOnboardingResponse);
    },
    enabled: Boolean(organizationId),
    staleTime: 0,
    refetchInterval: (current) => {
      if (options.poll === false) return false;
      return githubOnboardingPollInterval(current.state.data);
    },
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: githubOnboardingKey(organizationId) });
  const startInstallation = useMutation({
    mutationFn: async () => {
      const response = await meStartVcsProviderInstallation(
        withOrganizationHeader({ organizationId, path: { provider: githubProvider }, body: {} }),
      );
      if (!response.data?.url) throw new Error("GitHub did not return an installation URL");
      return response.data.url;
    },
  });
  const selectIdentity = useMutation({
    mutationFn: async (userId: string) => {
      await meSelectVcsProviderOnboardingIdentity(
        withOrganizationHeader({ organizationId, path: { provider: githubProvider }, body: { userId } }),
      );
    },
    onSuccess: () => invalidate(),
  });
  const configureInstallation = useMutation({
    mutationFn: async (installationId: string) => {
      const response = await meConfigureVcsProviderInstallation(
        withOrganizationHeader({
          organizationId,
          path: { provider: githubProvider, installationId },
          body: {},
        }),
      );
      if (!response.data?.url) throw new Error("GitHub did not return an installation settings URL");
      return response.data.url;
    },
  });
  const refresh = useMutation({
    mutationFn: async (repositoryId?: string) => {
      await meRefreshVcsProviderOnboarding(
        withOrganizationHeader({
          organizationId,
          path: { provider: githubProvider },
          body: { repositoryId },
        }),
      );
    },
    onSuccess: () => void invalidate(),
  });

  return {
    ...query,
    startInstallation,
    selectIdentity,
    configureInstallation,
    refresh,
  };
}

/**
 * GitHub does not always send a webhook when an App is uninstalled. While the
 * organization or repository choice is open, the server checks the visible
 * installations with GitHub, so an uninstalled organization leaves the list.
 */
export function useGitHubInstallationChecks(organizationId: string, enabled: boolean) {
  const queryClient = useQueryClient();
  useQuery({
    queryKey: githubInstallationChecksKey(organizationId),
    queryFn: async () => {
      await meVerifyVcsProviderInstallations(
        withOrganizationHeader({ organizationId, path: { provider: githubProvider }, body: {} }),
      );
      await queryClient.invalidateQueries({ queryKey: githubOnboardingKey(organizationId) });
      return null;
    },
    enabled: enabled && Boolean(organizationId),
    refetchInterval: githubInstallationCheckIntervalMs,
    retry: false,
    gcTime: 0,
  });
}
