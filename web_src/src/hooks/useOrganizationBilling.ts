import { useMutation, useQuery } from "@tanstack/react-query";

import { organizationsCreateBusinessCheckout, organizationsDescribeOrganizationBilling } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function organizationBillingQueryKey(organizationId: string) {
  return ["organizations", organizationId, "billing"] as const;
}

export function useOrganizationBilling(organizationId: string | undefined) {
  return useQuery({
    queryKey: organizationBillingQueryKey(organizationId ?? ""),
    queryFn: async () => {
      const response = await organizationsDescribeOrganizationBilling(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId),
    staleTime: 30 * 1000,
  });
}

export function useCreateBusinessCheckout(organizationId: string) {
  return useMutation({
    mutationFn: async () => {
      const response = await organizationsCreateBusinessCheckout(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: {},
        }),
      );
      const checkoutUrl = response.data?.checkoutUrl;
      if (!checkoutUrl) {
        throw new Error("Checkout is not available");
      }
      return checkoutUrl;
    },
  });
}
