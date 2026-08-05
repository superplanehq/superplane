import {
  factoriesCloseWorkOrder,
  factoriesCreateFactory,
  factoriesCreateFactoryLine,
  factoriesCreateWorkOrder,
  factoriesDescribeFactory,
  factoriesDescribeWorkOrder,
  factoriesDispatchWorkOrder,
  factoriesListFactories,
  factoriesListFactoryApps,
  factoriesListWorkOrderEvents,
  factoriesListWorkOrders,
  factoriesUpdateFactory,
  factoriesUpdateFactoryLine,
  factoriesUpdateWorkOrderAssignees,
} from "@/api-client";
import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderResult,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { hasActiveWorkOrderExecution } from "@/pages/factories/lib/workOrderExecutions";
import {
  getWorkOrderEventsNextPageParam,
  WORK_ORDER_EVENTS_PAGE_LIMIT,
} from "@/pages/factories/lib/workOrderEventsPagination";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const WORK_ORDER_POLL_INTERVAL_MS = 5_000;

function factoryListKey(organizationId: string) {
  return ["factories", organizationId] as const;
}

function factoryDetailKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId] as const;
}

function workOrdersKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId, "work-orders"] as const;
}

function workOrderDetailKey(organizationId: string, factoryId: string, orderId: string) {
  return ["factories", organizationId, factoryId, "work-orders", orderId] as const;
}

function workOrderEventsKey(organizationId: string, factoryId: string, orderId: string) {
  return ["factories", organizationId, factoryId, "work-orders", orderId, "events"] as const;
}

export function factoryAppsKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId, "apps"] as const;
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
    refetchInterval: (query) =>
      query.state.data?.some((order) => hasActiveWorkOrderExecution(order.executions))
        ? WORK_ORDER_POLL_INTERVAL_MS
        : false,
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
    refetchInterval: (query) =>
      query.state.data && hasActiveWorkOrderExecution(query.state.data.executions)
        ? WORK_ORDER_POLL_INTERVAL_MS
        : false,
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
    refetchInterval: WORK_ORDER_POLL_INTERVAL_MS,
  });
}

export function useCreateFactory(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name: string; description: string }) => {
      const response = await factoriesCreateFactory(
        withOrganizationHeader({
          organizationId,
          body: {
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

export function useUpdateFactory(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name?: string; description?: string }) => {
      const response = await factoriesUpdateFactory(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
          body: {
            name: input.name,
            description: input.description,
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
        throw new Error("Failed to create factory line");
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
        throw new Error("Failed to update factory line");
      }
      return response.data.line;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
    },
  });
}

export type { FactoryApp, FactoriesFactoryLine, FactoryLineStep };
