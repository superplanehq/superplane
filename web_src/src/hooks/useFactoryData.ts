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
  factoriesUpdateWorkOrder,
  factoriesUpdateWorkOrderAssignees,
  factoriesUpdateWorkOrderStatus,
  factoriesListFactoryPullRequests,
} from "@/api-client";
import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { markBacklogAnalysisPending } from "@/pages/factories/lib/backlogAnalysis";
import { buildOptimisticDispatchedOrder } from "@/pages/factories/lib/dispatchOptimistic";
import {
  getWorkOrderEventsNextPageParam,
  WORK_ORDER_EVENTS_PAGE_LIMIT,
} from "@/pages/factories/lib/workOrderEventsPagination";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export type FactoryPullRequestFilters = {
  order?: number | string;
  workOrderIds?: string[];
};

type NormalizedFactoryPullRequestFilters = {
  order?: string;
  workOrderIds: string[];
};

function normalizeFactoryPullRequestFilters(filters?: FactoryPullRequestFilters): NormalizedFactoryPullRequestFilters {
  const workOrderIds = [...new Set(filters?.workOrderIds ?? [])].filter(Boolean).sort();
  const order = filters?.order == null || String(filters.order) === "" ? undefined : String(filters.order);
  return { order, workOrderIds };
}

export const factoryQueryKeys = {
  list: (organizationId: string) => ["factories", organizationId] as const,
  detail: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId] as const,
  workOrders: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "work-orders"] as const,
  workOrderDetail: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId] as const,
  workOrderEvents: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "events"] as const,
  workOrderArtifacts: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "artifacts"] as const,
  workOrderChecks: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "checks"] as const,
  pullRequests: (organizationId: string, factoryId: string, filters: NormalizedFactoryPullRequestFilters) =>
    ["factories", organizationId, factoryId, "pull-requests", filters.order ?? "", ...filters.workOrderIds] as const,
  apps: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId, "apps"] as const,
  velocity: (
    organizationId: string,
    factoryId: string,
    params: {
      periodDays: number;
      repository: string;
      peopleSort: string;
      peopleSortDirection: string;
      peopleOffset: number;
      peoplePageSize: number;
    },
  ) =>
    [
      "factories",
      organizationId,
      factoryId,
      "velocity",
      params.periodDays,
      params.repository,
      params.peopleSort,
      params.peopleSortDirection,
      params.peopleOffset,
      params.peoplePageSize,
    ] as const,
  /** Every period and repository of one workspace, for refreshing after a sync. */
  velocityAll: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "velocity"] as const,
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
  return factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId);
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

export function factoryPullRequestsKey(organizationId: string, factoryId: string, filters?: FactoryPullRequestFilters) {
  return factoryQueryKeys.pullRequests(organizationId, factoryId, normalizeFactoryPullRequestFilters(filters));
}

export function useFactoryPullRequests(organizationId: string, factoryId: string, filters?: FactoryPullRequestFilters) {
  const normalized = normalizeFactoryPullRequestFilters(filters);
  return useQuery({
    queryKey: factoryQueryKeys.pullRequests(organizationId, factoryId, normalized),
    queryFn: async (): Promise<FactoriesFactoryPullRequest[]> => {
      const response = await factoriesListFactoryPullRequests(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: {
            order: normalized.order,
            workOrderIds: normalized.workOrderIds.length > 0 ? normalized.workOrderIds : undefined,
          },
        }),
      );
      return response.data?.pullRequests ?? [];
    },
    enabled: Boolean(organizationId && factoryId),
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
        throw new Error("Task not found");
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
    mutationFn: async (input: {
      name?: string;
      description?: string;
      key?: string;
      hostedSpendBudgetCents?: number | null;
    }) => {
      const response = await factoriesUpdateFactory(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
          body: {
            name: input.name,
            description: input.description,
            key: input.key,
            hostedSpendBudgetCents:
              input.hostedSpendBudgetCents === null || input.hostedSpendBudgetCents === undefined
                ? undefined
                : String(input.hostedSpendBudgetCents),
            clearHostedSpendBudget: input.hostedSpendBudgetCents === null ? true : undefined,
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
        throw new Error("Failed to create task");
      }
      return response.data.order;
    },
    onSuccess: (order) => {
      void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
      // The Backlog run for this order is created asynchronously after this
      // RPC returns, so show "Analyzing" optimistically and start polling
      // for the real run right away instead of waiting for a page reload.
      markBacklogAnalysisPending(order.id);
      void queryClient.invalidateQueries({ queryKey: ["backlog-analysis-runs", organizationId] });
      if (order.id) {
        void queryClient.invalidateQueries({
          queryKey: workOrderEventsKey(organizationId, factoryId, order.id),
        });
      }
    },
  });
}

export function useUpdateWorkOrder(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; title?: string; description?: string }) => {
      const response = await factoriesUpdateWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            title: input.title,
            description: input.description,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to update task");
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
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, variables.orderId),
      });
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
        throw new Error("Failed to update task assignees");
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
    mutationFn: async (input: {
      orderId: string;
      lineName: string;
      startStepIndex?: number;
      replaceActive?: boolean;
    }) => {
      const response = await factoriesDispatchWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            lineName: input.lineName,
            startStepIndex: input.startStepIndex,
            replaceActive: input.replaceActive,
          },
        }),
      );
      if (!response.data?.order) {
        throw new Error("Failed to dispatch task");
      }
      return response.data.order;
    },
    // Dispatch is synchronous on the server, but the UI would otherwise wait
    // for the whole-factory ListWorkOrders refetch triggered by onSuccess
    // before the card leaves Backlog. Patch the cache immediately instead, so
    // the card is already on the line's first phase column when this
    // function returns, and roll the patch back if the request fails.
    onMutate: async (variables) => {
      const ordersKey = workOrdersKey(organizationId, factoryId);
      await queryClient.cancelQueries({ queryKey: ordersKey });

      const previousOrders = queryClient.getQueryData<FactoriesWorkOrder[]>(ordersKey);
      const factory = queryClient.getQueryData<FactoriesFactory>(factoryDetailKey(organizationId, factoryId));
      const line = factory?.lines?.find((candidate) => candidate.name === variables.lineName);

      // Placing the card needs the line's id (and its steps, for the phase
      // label); skip the patch when the factory detail isn't cached yet.
      // Dispatch still succeeds — the card just waits for the invalidated
      // list below to move it, same as before this change.
      if (line && previousOrders) {
        const now = new Date().toISOString();
        queryClient.setQueryData<FactoriesWorkOrder[]>(ordersKey, (current) =>
          (current ?? []).map((order) =>
            order.id === variables.orderId ? buildOptimisticDispatchedOrder(order, line, now) : order,
          ),
        );
      }

      return { previousOrders };
    },
    onError: (_error, _variables, context) => {
      if (context?.previousOrders) {
        queryClient.setQueryData(workOrdersKey(organizationId, factoryId), context.previousOrders);
      }
    },
    onSuccess: (order, variables) => {
      // Reconcile the optimistic placeholder with the real dispatch id,
      // execution id, and run refs, so the card doesn't flicker back to
      // Backlog before the invalidated queries below refetch.
      queryClient.setQueryData<FactoriesWorkOrder[]>(workOrdersKey(organizationId, factoryId), (current) =>
        (current ?? []).map((existing) => (existing.id === order.id ? order : existing)),
      );
      queryClient.setQueryData(workOrderDetailKey(organizationId, factoryId, variables.orderId), order);
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
        throw new Error("Failed to update task status");
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
    mutationFn: async (input: { orderId: string; body: string; mentionedUserIds?: string[] }) => {
      const response = await factoriesAddWorkOrderComment(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            body: input.body,
            mentionedUserIds: input.mentionedUserIds,
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
        throw new Error("Failed to close task");
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

export async function fetchFactoryApps(organizationId: string, factoryId: string): Promise<FactoryApp[]> {
  const response = await factoriesListFactoryApps(
    withOrganizationHeader({
      organizationId,
      path: { factoryId },
    }),
  );
  return response.data?.apps ?? [];
}

export function useFactoryApps(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryAppsKey(organizationId, factoryId),
    queryFn: () => fetchFactoryApps(organizationId, factoryId),
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
    mutationFn: async (input: {
      lineId: string;
      name?: string;
      steps?: FactoryLineStep[];
      columnColors?: Record<string, string>;
    }) => {
      const response = await factoriesUpdateFactoryLine(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, lineId: input.lineId },
          body: {
            name: input.name,
            steps: input.steps,
            columnColors: input.columnColors,
          },
        }),
      );
      if (!response.data?.line) {
        throw new Error("Failed to update line");
      }
      return response.data.line;
    },
    onSuccess: (line) => {
      queryClient.setQueryData<FactoriesFactory>(factoryDetailKey(organizationId, factoryId), (current) => {
        if (!current?.lines) {
          return current;
        }
        return {
          ...current,
          lines: current.lines.map((existing) => (existing.id === line.id ? line : existing)),
        };
      });
      void queryClient.invalidateQueries({ queryKey: factoryDetailKey(organizationId, factoryId) });
    },
  });
}

export type { FactoryApp, FactoriesFactoryLine, FactoryLineStep };
