import { useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  organizationsCancelOrganizationSubscription,
  organizationsCreateBusinessCheckout,
  organizationsDescribeOrganizationBilling,
  organizationsResumeOrganizationSubscription,
  organizationsSyncOrganizationBilling,
} from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function organizationBillingQueryKey(organizationId: string) {
  return ["organizations", organizationId, "billing"] as const;
}

export async function syncOrganizationBilling(organizationId: string) {
  await organizationsSyncOrganizationBilling(
    withOrganizationHeader({
      organizationId,
      path: { id: organizationId },
      body: {},
    }),
  );
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

export function useCancelOrganizationSubscription(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const response = await organizationsCancelOrganizationSubscription(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: {},
        }),
      );
      return response.data ?? {};
    },
    onSuccess: (data) => {
      queryClient.setQueryData(organizationBillingQueryKey(organizationId), data);
    },
  });
}

export function useResumeOrganizationSubscription(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const response = await organizationsResumeOrganizationSubscription(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: {},
        }),
      );
      return response.data ?? {};
    },
    onSuccess: (data) => {
      queryClient.setQueryData(organizationBillingQueryKey(organizationId), data);
    },
  });
}

export function useOrganizationSubscriptionActions(organizationId: string) {
  const cancelSubscription = useCancelOrganizationSubscription(organizationId);
  const resumeSubscription = useResumeOrganizationSubscription(organizationId);

  const onCancelSubscription = useCallback(async () => {
    try {
      await cancelSubscription.mutateAsync();
    } catch (cancelError) {
      showErrorToast(getApiErrorMessage(cancelError, "Unable to cancel Business."));
      throw cancelError;
    }
  }, [cancelSubscription]);

  const onKeepSubscription = useCallback(async () => {
    try {
      await resumeSubscription.mutateAsync();
    } catch (keepError) {
      showErrorToast(getApiErrorMessage(keepError, "Unable to keep Business."));
      throw keepError;
    }
  }, [resumeSubscription]);

  return {
    cancelPending: cancelSubscription.isPending,
    keepPending: resumeSubscription.isPending,
    onCancelSubscription,
    onKeepSubscription,
  };
}
