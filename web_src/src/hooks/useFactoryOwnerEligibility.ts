import { useQuery } from "@tanstack/react-query";
import { factoriesListFactoryOwnerEligibility } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function factoryOwnerEligibilityKey(organizationId: string, factoryId: string, orderId = "", memberId = "") {
  return ["factories", organizationId, factoryId, "owner-eligibility", orderId, memberId] as const;
}

export function useFactoryOwnerEligibility(
  organizationId: string,
  factoryId: string,
  orderId?: string,
  prospectiveSubscription = false,
) {
  const { currentUserId } = usePermissions();
  return useQuery({
    queryKey: [
      ...factoryOwnerEligibilityKey(organizationId, factoryId, orderId, currentUserId),
      prospectiveSubscription,
    ],
    queryFn: async () => {
      const response = await factoriesListFactoryOwnerEligibility(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { orderId, prospectiveSubscription },
        }),
      );
      if (!response.data) throw new Error("Could not load owner eligibility.");
      return response.data;
    },
    enabled: Boolean(organizationId && factoryId),
    staleTime: 15000,
  });
}
