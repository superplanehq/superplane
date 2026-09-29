import {
  meConfigureVcsProviderInstallation,
  meDescribeVcsProviderOnboarding,
  meRefreshVcsProviderOnboarding,
  meSelectVcsProviderOnboardingIdentity,
  meStartVcsProviderInstallation,
  type MeDescribeVcsProviderOnboardingResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const githubOnboardingKey = (organizationId: string) => ["me", organizationId, "github-onboarding"] as const;
const githubProvider = "github";

export function useGitHubOnboarding(organizationId: string) {
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
      const data = current.state.data;
      return data?.identity ? 3_000 : false;
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
