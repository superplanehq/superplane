import {
  factoriesCreateFactoryCustomVerification,
  factoriesDescribeFactoryCustomVerification,
  factoriesRunFactoryCustomVerification,
  factoriesUpdateFactoryCustomVerification,
  type FactoriesFactoryCustomVerification,
  type FactoriesFactoryCustomVerificationSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryAppsKey, factoryQueryKeys } from "./useFactoryData";

export const factoryCustomVerificationKey = (organizationId: string, factoryId: string, automationId: string) =>
  ["factories", organizationId, factoryId, "custom-verifications", automationId] as const;

async function fetchFactoryCustomVerification(
  organizationId: string,
  factoryId: string,
  automationId: string,
): Promise<FactoriesFactoryCustomVerification> {
  const response = await factoriesDescribeFactoryCustomVerification(
    withOrganizationHeader({
      organizationId,
      path: { factoryId, automationId },
    }),
  );
  if (!response.data?.verification) {
    throw new Error("Automation settings are unavailable.");
  }
  return response.data.verification;
}

export function useFactoryCustomVerification(
  organizationId: string,
  factoryId: string,
  automationId: string | undefined,
) {
  return useQuery({
    queryKey: factoryCustomVerificationKey(organizationId, factoryId, automationId ?? ""),
    queryFn: () => fetchFactoryCustomVerification(organizationId, factoryId, automationId ?? ""),
    enabled: Boolean(organizationId && factoryId && automationId),
  });
}

export function useFactoryCustomVerifications(
  organizationId: string,
  factoryId: string,
  automationIds: readonly string[],
) {
  const queries = useQueries({
    queries: automationIds.map((automationId) => ({
      queryKey: factoryCustomVerificationKey(organizationId, factoryId, automationId),
      queryFn: () => fetchFactoryCustomVerification(organizationId, factoryId, automationId),
      enabled: Boolean(organizationId && factoryId && automationId),
      staleTime: 30_000,
    })),
  });

  return automationIds.flatMap((automationId, index) => {
    const verification = queries[index]?.data;
    return verification ? [{ automationId, verification }] : [];
  });
}

type CustomVerificationInput = {
  name: string;
  settings: FactoriesFactoryCustomVerificationSettings;
};

export function useCreateFactoryCustomVerification(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CustomVerificationInput) => {
      const response = await factoriesCreateFactoryCustomVerification(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: input,
        }),
      );
      if (!response.data?.verification) {
        throw new Error("SuperPlane could not create the automation.");
      }
      return response.data.verification;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useUpdateFactoryCustomVerification(organizationId: string, factoryId: string, automationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CustomVerificationInput) => {
      const response = await factoriesUpdateFactoryCustomVerification(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, automationId },
          body: input,
        }),
      );
      if (!response.data?.verification) {
        throw new Error("SuperPlane could not update the automation.");
      }
      return response.data.verification;
    },
    onSuccess: (verification) => {
      queryClient.setQueryData(factoryCustomVerificationKey(organizationId, factoryId, automationId), verification);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useRunFactoryCustomVerification(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ automationId, workOrderId }: { automationId: string; workOrderId: string }) => {
      const response = await factoriesRunFactoryCustomVerification(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, automationId },
          body: { workOrderId },
        }),
      );
      if (!response.data?.runId) {
        throw new Error("SuperPlane could not start the automation.");
      }
      return response.data;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, variables.workOrderId),
      });
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
    },
  });
}
