import {
  factoriesAddWorkOrderComment,
  factoriesCloseWorkOrder,
  factoriesCreateFactory,
  factoriesCreateFactoryLine,
  factoriesCreateWorkOrder,
  factoriesDeleteFactory,
  factoriesDescribeFactory,
  factoriesDescribeWorkOrder,
  factoriesDispatchWorkOrder,
  factoriesListFactories,
  factoriesListFactoryApps,
  factoriesListWorkOrderArtifacts,
  factoriesListWorkOrderEvents,
  factoriesListWorkOrders,
  factoriesUpdateFactory,
  factoriesUpdateFactoryLine,
  factoriesUpdateWorkOrderAssignees,
  factoriesUpdateWorkOrderStatus,
} from "@/api-client";
import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  getWorkOrderEventsNextPageParam,
  WORK_ORDER_EVENTS_PAGE_LIMIT,
} from "@/pages/factories/lib/workOrderEventsPagination";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const factoryQueryKeys = {
  list: (organizationId: string) => ["factories", organizationId] as const,
  detail: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId] as const,
  workOrders: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "work-orders"] as const,
  workOrderDetail: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId] as const,
  workOrderEvents: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "events"] as const,
  apps: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId, "apps"] as const,
};

function factoryListKey(organizationId: string) {
  return factoryQueryKeys.list(organizationId);
}

function factoryDetailKey(organizationId: string, factoryId: string) {
  return factoryQueryKeys.detail(organizationId, factoryId);
}

function workOrdersKey(organizationId: string, factoryId: string) {
  return factoryQueryKeys.workOrders(organizationId, factoryId);
}

function workOrderDetailKey(organizationId: string, factoryId: string, orderId: string) {
  return factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId);
}

function workOrderEventsKey(organizationId: string, factoryId: string, orderId: string) {
  return factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId);
}

function workOrderArtifactsKey(organizationId: string, factoryId: string, orderId: string) {
  return ["factories", organizationId, factoryId, "work-orders", orderId, "artifacts"] as const;
}

export function factoryAppsKey(organizationId: string, factoryId: string) {
  return factoryQueryKeys.apps(organizationId, factoryId);
}

export function useFactories(organizationId: string, enabled = true) {
  return useQuery({
    queryKey: factoryListKey(organizationId),
    queryFn: async (): Promise<FactoriesFactory[]> => {
      const response = await factoriesListFactories(withOrganizationHeader({ organizationId }));
      return response.data?.factories ?? [];
    },
    enabled: Boolean(organizationId) && enabled,
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
    // Live via websocket; remount must refetch instead of serving 5m global stale cache.
    staleTime: 0,
  });
}

export function useWorkOrder(organizationId: string, factoryId: string, orderId: string) {
  return useQuery({
    queryKey: workOrderDetailKey(organizationId, factoryId, orderId),
    queryFn: async (): Promise<FactoriesWorkOrder> => {
      const response = await factoriesDescribeWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Work order not found");
      }
      return response.data.order;
    },
    enabled: Boolean(organizationId && factoryId && orderId),
    staleTime: 0,
  });
}

export function useWorkOrderEvents(organizationId: string, factoryId: string, orderId: string) {
  return useInfiniteQuery({
    queryKey: workOrderEventsKey(organizationId, factoryId, orderId),
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const response = await factoriesListWorkOrderEvents(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId },
          query: {
            limit: WORK_ORDER_EVENTS_PAGE_LIMIT,
            ...(pageParam ? { before: pageParam } : {}),
          },
        }),
      );
      return response.data;
    },
    getNextPageParam: getWorkOrderEventsNextPageParam,
    initialPageParam: undefined as string | undefined,
    enabled: Boolean(organizationId && factoryId && orderId),
    staleTime: 0,
  });
}

export function useCreateFactory(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; description: string; key: string }) => {
      const response = await factoriesCreateFactory(
        withOrganizationHeader({
          organizationId,
          body: {
            name: input.name,
            description: input.description,
            key: input.key,
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

export function useUpdateFactory(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name?: string; description?: string; key?: string }) => {
      const response = await factoriesUpdateFactory(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
          body: {
            name: input.name,
            description: input.description,
            key: input.key,
          },
        }),
      );
      if (!response.data?.factory) {
        throw new Error("Failed to update factory");
      }
      return response.data.factory;
    },
    onSuccess: (factory) => {
      queryClient.setQueryData(factoryDetailKey(organizationId, factoryId), factory);
      void queryClient.invalidateQueries({ queryKey: factoryListKey(organizationId) });
      void queryClient.invalidateQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
    },
  });
}

export function useDeleteFactory(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (factoryId: string) => {
      await factoriesDeleteFactory(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
        }),
      );
      return factoryId;
    },
    onSuccess: (factoryId) => {
      queryClient.setQueryData<FactoriesFactory[]>(factoryListKey(organizationId), (current) =>
        (current ?? []).filter((factory) => factory.id !== factoryId),
      );
      queryClient.removeQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryListKey(organizationId) });
    },
  });
}

export function useCreateWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { title: string; description: string; assigneeIds?: string[] }) => {
      const response = await factoriesCreateWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            title: input.title,
            description: input.description,
            assigneeIds: input.assigneeIds,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to create work order");
      }
      return response.data.order;
    },
    onSuccess: (order) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      if (order.id) {
        void queryClient.invalidateQueries({
          queryKey: workOrderEventsKey(organizationId, factoryId, order.id),
        });
      }
    },
  });
}

export function useUpdateWorkOrderAssignees(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; assigneeIds: string[] }) => {
      const response = await factoriesUpdateWorkOrderAssignees(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            assigneeIds: input.assigneeIds,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to update work order assignees");
      }
      return response.data.order;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: workOrderDetailKey(organizationId, factoryId, variables.orderId),
      });
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export function useDispatchWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; lineName: string }) => {
      const response = await factoriesDispatchWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            lineName: input.lineName,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to dispatch work order");
      }
      return response.data.order;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: workOrderDetailKey(organizationId, factoryId, variables.orderId),
      });
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export function useUpdateWorkOrderStatus(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      orderId: string;
      state: FactoriesWorkOrderState;
      result?: FactoriesWorkOrderResult;
    }) => {
      const response = await factoriesUpdateWorkOrderStatus(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            state: input.state,
            result: input.result,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to update work order status");
      }
      return response.data.order;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: workOrderDetailKey(organizationId, factoryId, variables.orderId),
      });
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export function useAddWorkOrderComment(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; body: string }) => {
      const response = await factoriesAddWorkOrderComment(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            body: input.body,
          },
        }),
      );
      if (!response.data?.comment) {
        throw new Error("Failed to add comment");
      }
      return response.data.comment;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export function useWorkOrderArtifacts(organizationId: string, factoryId: string, orderId: string) {
  return useQuery({
    queryKey: workOrderArtifactsKey(organizationId, factoryId, orderId),
    queryFn: async (): Promise<FactoriesWorkOrderArtifact[]> => {
      const response = await factoriesListWorkOrderArtifacts(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId },
        }),
      );
      return response.data?.artifacts ?? [];
    },
    enabled: Boolean(organizationId && factoryId && orderId),
  });
}

export function useCloseWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; result: FactoriesWorkOrderResult }) => {
      const response = await factoriesCloseWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            result: input.result,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to close work order");
      }
      return response.data.order;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: workOrderDetailKey(organizationId, factoryId, variables.orderId),
      });
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export function useFactoryApps(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryAppsKey(organizationId, factoryId),
    queryFn: async (): Promise<FactoryApp[]> => {
      const response = await factoriesListFactoryApps(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      return response.data?.apps ?? [];
    },
    enabled: Boolean(organizationId && factoryId),
  });
}

export function useCreateFactoryLine(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; steps: FactoryLineStep[] }) => {
      const response = await factoriesCreateFactoryLine(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            name: input.name,
            steps: input.steps,
          },
        }),
      );
      if (!response.data?.line) {
        throw new Error("Failed to create line");
      }
      return response.data.line;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
    },
  });
}

export function useUpdateFactoryLine(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { lineId: string; name?: string; steps?: FactoryLineStep[] }) => {
      const response = await factoriesUpdateFactoryLine(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, lineId: input.lineId },
          body: {
            name: input.name,
            steps: input.steps,
          },
        }),
      );
      if (!response.data?.line) {
        throw new Error("Failed to update line");
      }
      return response.data.line;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
    },
  });
}

export type { FactoryApp, FactoriesFactoryLine, FactoryLineStep };
