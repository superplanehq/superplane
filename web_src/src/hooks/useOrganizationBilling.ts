import { useCallback } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import {
  organizationsCancelOrganizationSubscription,
  organizationsDescribeOrganizationBilling,
  organizationsResumeOrganizationSubscription,
  organizationsSyncOrganizationBilling,
  type OrganizationsDescribeOrganizationBillingResponse,
} from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function organizationBillingQueryKey(organizationId: string) {
  return ["organizations", organizationId, "billing"] as const;
}

const billingSynchronizations = new WeakMap<
  QueryClient,
  Map<string, Promise<OrganizationsDescribeOrganizationBillingResponse>>
>();

export function syncOrganizationBilling(organizationId: string, queryClient: QueryClient) {
  let synchronizations = billingSynchronizations.get(queryClient);
  if (!synchronizations) {
    synchronizations = new Map();
    billingSynchronizations.set(queryClient, synchronizations);
  }
  const pending = synchronizations.get(organizationId);
  if (pending) return pending;

  const synchronize = async () => {
    const response = await organizationsSyncOrganizationBilling(
      withOrganizationHeader({
        organizationId,
        path: { id: organizationId },
        body: {},
      }),
    );
    if (!response.data) throw new Error("SuperPlane could not synchronize billing. Try again.");
    queryClient.setQueryData(organizationBillingQueryKey(organizationId), response.data);
    await invalidateOrganizationModelAccess(queryClient, organizationId);
    return response.data;
  };
  const synchronization = synchronize().finally(() => synchronizations.delete(organizationId));
  synchronizations.set(organizationId, synchronization);
  return synchronization;
}

export function invalidateOrganizationModelAccess(queryClient: QueryClient, organizationId: string) {
  return queryClient.invalidateQueries({
    predicate: (query) =>
      query.queryKey[1] === organizationId &&
      ((query.queryKey[0] === "organizations" && query.queryKey[2] !== "billing") || query.queryKey[0] === "factories"),
  });
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
      showErrorToast(getApiErrorMessage(cancelError, "Unable to cancel subscription."));
      throw cancelError;
    }
  }, [cancelSubscription]);

  const onKeepSubscription = useCallback(async () => {
    try {
      await resumeSubscription.mutateAsync();
    } catch (keepError) {
      showErrorToast(getApiErrorMessage(keepError, "Unable to keep subscription."));
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
