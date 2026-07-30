import {
  factoriesAssignWorkOrder,
  factoriesCreateFactory,
  factoriesCreateWorkOrder,
  factoriesDescribeFactory,
  factoriesListFactories,
  factoriesListWorkOrders,
} from "@/api-client";
import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

function factoryListKey(organizationId: string) {
  return ["factories", organizationId] as const;
}

function factoryDetailKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId] as const;
}

function workOrdersKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId, "work-orders"] as const;
}

export function useFactories(organizationId: string) {
  return useQuery({
    queryKey: factoryListKey(organizationId),
    queryFn: async (): Promise<FactoriesFactory[]> => {
      const response = await factoriesListFactories(withOrganizationHeader({ organizationId }));
      return response.data?.factories ?? [];
    },
    enabled: Boolean(organizationId),
  });
}

export function useFactory(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryDetailKey(organizationId, factoryId),
    queryFn: async (): Promise<FactoriesFactory> => {
      const response = await factoriesDescribeFactory(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
        }),
      );
      if (!response.data?.factory) {
        throw new Error("Factory not found");
      }
      return response.data.factory;
    },
    enabled: Boolean(organizationId && factoryId),
  });
}

export function useFactoryWorkOrders(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: workOrdersKey(organizationId, factoryId),
    queryFn: async (): Promise<FactoriesWorkOrder[]> => {
      const response = await factoriesListWorkOrders(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      return response.data?.orders ?? [];
    },
    enabled: Boolean(organizationId && factoryId),
  });
}

export function useCreateFactory(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; description: string }) => {
      const response = await factoriesCreateFactory(
        withOrganizationHeader({
          organizationId,
          query: {
            name: input.name,
            description: input.description,
          },
        }),
      );
      if (!response.data?.factory) {
        throw new Error("Failed to create factory");
      }
      return response.data.factory;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryListKey(organizationId) });
    },
  });
}

export function useCreateWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { title: string; description: string }) => {
      const response = await factoriesCreateWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            title: input.title,
            description: input.description,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to create work order");
      }
      return response.data.order;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
    },
  });
}

export function useAssignWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; assigneeIds: string[] }) => {
      const response = await factoriesAssignWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            assigneeIds: input.assigneeIds,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to assign work order");
      }
      return response.data.order;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
    },
  });
}
