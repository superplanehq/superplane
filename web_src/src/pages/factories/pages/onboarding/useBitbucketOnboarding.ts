import {
  meDescribeVcsProviderOnboarding,
  meSelectVcsProviderOnboardingIdentity,
  meStartVcsProviderInstallation,
  type MeDescribeVcsProviderOnboardingResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { bitbucketOnboardingPollInterval } from "./bitbucketOnboardingPoll";

const bitbucketOnboardingKey = (organizationId: string) => ["me", organizationId, "bitbucket-onboarding"] as const;
const bitbucketProvider = "bitbucket";

export function useBitbucketOnboarding(
  organizationId: string,
  options: { poll?: boolean; attemptActive?: boolean } = {},
) {
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
    refetchInterval: (current) =>
      bitbucketOnboardingPollInterval(options.poll !== false, current.state, options.attemptActive === true),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: bitbucketOnboardingKey(organizationId) });
  const startInstallation = useMutation({
    mutationFn: async () => {
      const response = await meStartVcsProviderInstallation(
        withOrganizationHeader({ organizationId, path: { provider: bitbucketProvider }, body: {} }),
      );
      if (!response.data?.url) throw new Error("Bitbucket did not return an installation URL");
      return response.data.url;
    },
    onSuccess: () => void invalidate(),
  });
  const selectIdentity = useMutation({
    mutationFn: async (providerUserId: string) => {
      await meSelectVcsProviderOnboardingIdentity(
        withOrganizationHeader({ organizationId, path: { provider: bitbucketProvider }, body: { providerUserId } }),
      );
    },
    onSuccess: () => invalidate(),
  });

  return { ...query, startInstallation, selectIdentity };
}
