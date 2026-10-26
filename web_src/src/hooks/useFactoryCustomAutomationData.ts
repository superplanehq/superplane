import {
  factoriesCreateFactoryCustomAutomation,
  factoriesDescribeFactoryCustomAutomation,
  factoriesRunFactoryCustomAutomation,
  factoriesUpdateFactoryCustomAutomation,
  type FactoriesFactoryCustomAutomation,
  type FactoriesFactoryCustomAutomationSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryAppsKey, factoryQueryKeys } from "./useFactoryData";

export const factoryCustomAutomationKey = (organizationId: string, factoryId: string, automationId: string) =>
  ["factories", organizationId, factoryId, "custom-automations", automationId] as const;

async function fetchFactoryCustomAutomation(
  organizationId: string,
  factoryId: string,
  automationId: string,
): Promise<FactoriesFactoryCustomAutomation> {
  const response = await factoriesDescribeFactoryCustomAutomation(
    withOrganizationHeader({
      organizationId,
      path: { factoryId, automationId },
    }),
  );
  if (!response.data?.customAutomation) {
    throw new Error("Automation settings are unavailable.");
  }
  return response.data.customAutomation;
}

export function useFactoryCustomAutomation(
  organizationId: string,
  factoryId: string,
  automationId: string | undefined,
) {
  return useQuery({
    queryKey: factoryCustomAutomationKey(organizationId, factoryId, automationId ?? ""),
    queryFn: () => fetchFactoryCustomAutomation(organizationId, factoryId, automationId ?? ""),
    enabled: Boolean(organizationId && factoryId && automationId),
  });
}

export function useFactoryCustomAutomations(
  organizationId: string,
  factoryId: string,
  automationIds: readonly string[],
) {
  const queries = useQueries({
    queries: automationIds.map((automationId) => ({
      queryKey: factoryCustomAutomationKey(organizationId, factoryId, automationId),
      queryFn: () => fetchFactoryCustomAutomation(organizationId, factoryId, automationId),
      enabled: Boolean(organizationId && factoryId && automationId),
      staleTime: 30_000,
    })),
  });

  return automationIds.flatMap((automationId, index) => {
    const automation = queries[index]?.data;
    return automation ? [{ automationId, automation }] : [];
  });
}

type CustomAutomationInput = {
  name: string;
  settings: FactoriesFactoryCustomAutomationSettings;
};

export function useCreateFactoryCustomAutomation(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CustomAutomationInput) => {
      const response = await factoriesCreateFactoryCustomAutomation(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: input,
        }),
      );
      if (!response.data?.customAutomation) {
        throw new Error("SuperPlane could not create the automation.");
      }
      return response.data.customAutomation;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useUpdateFactoryCustomAutomation(organizationId: string, factoryId: string, automationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CustomAutomationInput) => {
      const response = await factoriesUpdateFactoryCustomAutomation(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, automationId },
          body: input,
        }),
      );
      if (!response.data?.customAutomation) {
        throw new Error("SuperPlane could not update the automation.");
      }
      return response.data.customAutomation;
    },
    onSuccess: (automation) => {
      queryClient.setQueryData(factoryCustomAutomationKey(organizationId, factoryId, automationId), automation);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useRunFactoryCustomAutomation(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ automationId, workOrderId }: { automationId: string; workOrderId: string }) => {
      const response = await factoriesRunFactoryCustomAutomation(
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
