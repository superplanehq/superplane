import {
  meDescribeVcsProviderOnboarding,
  meStartVcsProviderInstallation,
  type MeDescribeVcsProviderOnboardingResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const bitbucketOnboardingKey = (organizationId: string) => ["me", organizationId, "bitbucket-onboarding"] as const;
const bitbucketProvider = "bitbucket";
const bitbucketOnboardingPollIntervalMs = 3_000;

export function useBitbucketOnboarding(organizationId: string, options: { poll?: boolean } = {}) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: bitbucketOnboardingKey(organizationId),
    queryFn: async () => {
      const response = await meDescribeVcsProviderOnboarding(
        withOrganizationHeader({ organizationId, path: { provider: bitbucketProvider } }),
      );
      return response.data ?? ({} as MeDescribeVcsProviderOnboardingResponse);
    },
    enabled: Boolean(organizationId),
    staleTime: 0,
    refetchInterval: (current) => {
      if (options.poll === false) return false;
      const data = current.state.data;
      if (!data?.providerConfigured || !data.identity) return false;
      if ((data.repositories ?? []).length > 0) return false;
      return bitbucketOnboardingPollIntervalMs;
    },
  });

  const startInstallation = useMutation({
    mutationFn: async () => {
      const response = await meStartVcsProviderInstallation(
        withOrganizationHeader({ organizationId, path: { provider: bitbucketProvider }, body: {} }),
      );
      if (!response.data?.url) throw new Error("Bitbucket did not return an installation URL");
      return response.data.url;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: bitbucketOnboardingKey(organizationId) });
    },
  });

  return { ...query, startInstallation };
}
