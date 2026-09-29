import {
  meConfigureGitHubAppInstallation,
  meDescribeGitHubOnboarding,
  meRefreshGitHubOnboarding,
  meSelectGitHubOnboardingIdentity,
  meStartGitHubAppInstallation,
  type MeDescribeGitHubOnboardingResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const githubOnboardingKey = (organizationId: string) => ["me", organizationId, "github-onboarding"] as const;

export function useGitHubOnboarding(organizationId: string) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: githubOnboardingKey(organizationId),
    queryFn: async () => {
      const response = await meDescribeGitHubOnboarding(withOrganizationHeader({ organizationId }));
      return response.data ?? ({} as MeDescribeGitHubOnboardingResponse);
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
      const response = await meStartGitHubAppInstallation(withOrganizationHeader({ organizationId, body: {} }));
      if (!response.data?.url) throw new Error("GitHub did not return an installation URL");
      return response.data.url;
    },
  });
  const selectIdentity = useMutation({
    mutationFn: async (userId: string) => {
      await meSelectGitHubOnboardingIdentity(withOrganizationHeader({ organizationId, body: { userId } }));
    },
    onSuccess: () => invalidate(),
  });
  const configureInstallation = useMutation({
    mutationFn: async (installationId: string) => {
      const response = await meConfigureGitHubAppInstallation(
        withOrganizationHeader({ organizationId, path: { installationId }, body: {} }),
      );
      if (!response.data?.url) throw new Error("GitHub did not return an installation settings URL");
      return response.data.url;
    },
  });
  const refresh = useMutation({
    mutationFn: async (repositoryId?: string) => {
      await meRefreshGitHubOnboarding(withOrganizationHeader({ organizationId, body: { repositoryId } }));
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
