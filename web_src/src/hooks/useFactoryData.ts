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
  factoriesCreateFactoryAutomation,
  factoriesDeleteFactoryAutomation,
  factoriesListFactories,
  factoriesListFactoryAutomations,
  factoriesListWorkOrderArtifacts,
  factoriesListWorkOrderEvents,
  factoriesListWorkOrders,
  factoriesUpdateFactory,
  factoriesUpdateFactoryLine,
  factoriesUpdateWorkOrder,
  factoriesUpdateWorkOrderAssignees,
  factoriesUpdateWorkOrderStatus,
} from "@/api-client";
import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderSummary,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
  FactoryAutomation,
  FactoryLineStep,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { markBacklogAnalysisPending } from "@/pages/factories/lib/backlogAnalysis";
import { buildOptimisticDispatchedOrder } from "@/pages/factories/lib/dispatchOptimistic";
import {
  getWorkOrderEventsNextPageParam,
  WORK_ORDER_EVENTS_PAGE_LIMIT,
} from "@/pages/factories/lib/workOrderEventsPagination";
import {
  BOARD_BACKLOG_PAGE_SIZE,
  BOARD_BACKLOG_STATES,
  BOARD_DONE_PAGE_SIZE,
  BOARD_DONE_STATES,
  BOARD_OPEN_PAGE_SIZE,
  BOARD_OPEN_STATES,
  factoryWorkOrdersPageKey,
  factoryWorkOrdersPagePrefix,
  flattenWorkOrdersPages,
  getWorkOrdersNextPageParam,
  normalizeWorkOrdersPageQuery,
  uniqueWorkOrdersById,
  WORK_ORDER_LIST_PAGE_SIZE,
  workOrdersPageFromResponse,
  type WorkOrdersPageCursor,
  type WorkOrdersPageQuery,
} from "@/pages/factories/lib/workOrderListPagination";
import { applyWorkOrderToListCaches, cachedWorkOrderFromLists } from "./workOrderListCache";
import { keepPreviousData, useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const factoryQueryKeys = {
  list: (organizationId: string) => ["factories", organizationId] as const,
  detail: (organizationId: string, factoryId: string) => ["factories", organizationId, factoryId] as const,
  workOrders: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "work-orders"] as const,
  workOrdersPagePrefix: factoryWorkOrdersPagePrefix,
  workOrdersPage: factoryWorkOrdersPageKey,
  workOrderDetail: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId] as const,
  workOrderEvents: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "events"] as const,
  workOrderArtifacts: (organizationId: string, factoryId: string, orderId: string) =>
    ["factories", organizationId, factoryId, "work-orders", orderId, "artifacts"] as const,
  planningSessions: (organizationId: string, factoryId: string) =>
    ["planning-session-by-work-order", organizationId, factoryId] as const,
  planningSession: (organizationId: string, factoryId: string, workOrderId: string) =>
    [...factoryQueryKeys.planningSessions(organizationId, factoryId), workOrderId] as const,
  pullRequestMergeability: (organizationId: string, factoryId: string, pullRequestId: string) =>
    ["factories", organizationId, factoryId, "pull-requests", pullRequestId, "mergeability"] as const,
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
    queryFn: async (): Promise<FactoriesWorkOrderSummary[]> => {
      const orders: FactoriesWorkOrderSummary[] = [];
      let beforeId: string | undefined;
      for (;;) {
        const response = await factoriesListWorkOrders(
          withOrganizationHeader({
            organizationId,
            path: { factoryId },
            query: {
              limit: WORK_ORDER_LIST_PAGE_SIZE,
              ...(beforeId ? { beforeId } : {}),
            },
          }),
        );
        const page = response.data?.orders ?? [];
        orders.push(...page);
        if (!response.data?.hasNextPage || page.length === 0) {
          return orders;
        }
        const nextId = page.at(-1)?.id;
        if (!nextId) {
          return orders;
        }
        beforeId = nextId;
      }
    },
    enabled: Boolean(organizationId && factoryId),
    // Live via websocket; remount must refetch instead of serving 5m global stale cache.
    staleTime: 0,
  });
}

export type FactoryWorkOrdersPageOptions = Partial<WorkOrdersPageQuery> & { requireUser?: boolean };

function workOrdersPageQueryFromOptions(options?: FactoryWorkOrdersPageOptions): WorkOrdersPageQuery {
  return normalizeWorkOrdersPageQuery({
    userId: options?.userId,
    unassigned: options?.unassigned,
  });
}

export function useFactoryWorkOrdersPage(
  organizationId: string,
  factoryId: string,
  states: readonly FactoriesWorkOrderState[],
  pageSize: number,
  options?: FactoryWorkOrdersPageOptions,
) {
  const pageQuery = workOrdersPageQueryFromOptions(options);
  const query = useInfiniteQuery({
    queryKey: factoryQueryKeys.workOrdersPage(organizationId, factoryId, states, pageQuery),
    queryFn: async ({ pageParam }: { pageParam?: WorkOrdersPageCursor }) => {
      const response = await factoriesListWorkOrders(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: {
            states: [...states],
            limit: pageSize,
            ...(pageQuery.userId ? { userId: pageQuery.userId } : {}),
            ...(pageQuery.unassigned ? { unassigned: true } : {}),
            ...(pageParam ? { beforeId: pageParam.beforeId } : {}),
          },
        }),
      );
      return workOrdersPageFromResponse(response.data);
    },
    getNextPageParam: getWorkOrdersNextPageParam,
    initialPageParam: undefined as WorkOrdersPageCursor | undefined,
    enabled: Boolean(organizationId && factoryId) && (!options?.requireUser || Boolean(pageQuery.userId)),
    staleTime: 0,
    placeholderData: keepPreviousData,
  });

  return {
    orders: flattenWorkOrdersPages(query.data?.pages),
    isLoading: query.isLoading,
    isPlaceholderData: query.isPlaceholderData,
    hasNextPage: Boolean(query.hasNextPage),
    fetchNextPage: query.fetchNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
  };
}

export type FactoryBoardColumnPage = {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
};

export type FactoryBoardWorkOrders = {
  workOrders: FactoriesWorkOrderSummary[];
  isLoading: boolean;
  isPlaceholderData: boolean;
  backlog: FactoryBoardColumnPage;
  open: FactoryBoardColumnPage;
  done: FactoryBoardColumnPage;
};

function boardColumnPage(page: ReturnType<typeof useFactoryWorkOrdersPage>): FactoryBoardColumnPage {
  return {
    hasNextPage: page.hasNextPage,
    isFetchingNextPage: page.isFetchingNextPage,
    fetchNextPage: () => {
      void page.fetchNextPage();
    },
  };
}

export function mergeFactoryBoardWorkOrders(
  backlog: FactoriesWorkOrderSummary[],
  open: FactoriesWorkOrderSummary[],
  done: FactoriesWorkOrderSummary[],
): FactoriesWorkOrderSummary[] {
  return uniqueWorkOrdersById([...backlog, ...open, ...done]);
}

export function useFactoryBoardWorkOrders(
  organizationId: string,
  factoryId: string,
  options?: FactoryWorkOrdersPageOptions,
): FactoryBoardWorkOrders {
  const backlog = useFactoryWorkOrdersPage(
    organizationId,
    factoryId,
    BOARD_BACKLOG_STATES,
    BOARD_BACKLOG_PAGE_SIZE,
    options,
  );
  const open = useFactoryWorkOrdersPage(organizationId, factoryId, BOARD_OPEN_STATES, BOARD_OPEN_PAGE_SIZE, options);
  const closed = useFactoryWorkOrdersPage(organizationId, factoryId, BOARD_DONE_STATES, BOARD_DONE_PAGE_SIZE, options);

  return {
    workOrders: mergeFactoryBoardWorkOrders(backlog.orders, open.orders, closed.orders),
    isLoading: backlog.isLoading || open.isLoading || closed.isLoading,
    isPlaceholderData: backlog.isPlaceholderData || open.isPlaceholderData || closed.isPlaceholderData,
    backlog: boardColumnPage(backlog),
    open: boardColumnPage(open),
    done: boardColumnPage(closed),
  };
}

function invalidateWorkOrderLists(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
) {
  void queryClient.invalidateQueries({ queryKey: workOrdersKey(organizationId, factoryId) });
  void queryClient.invalidateQueries({ queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId) });
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
      planning?: { enabled: boolean; clarity: boolean; confidence: boolean; setupCompleted?: boolean };
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
            planning: input.planning,
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
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
      // The Backlog run for this order is created asynchronously after this
      // RPC returns. Show "Analyzing" until the canvas WebSocket delivers it.
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
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
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
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
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
      model?: string;
    }) => {
      const response = await factoriesDispatchWorkOrder(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: {
            lineName: input.lineName,
            startStepIndex: input.startStepIndex,
            replaceActive: input.replaceActive,
            model: input.model,
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
      const pagePrefix = factoryWorkOrdersPagePrefix(organizationId, factoryId);
      await queryClient.cancelQueries({ queryKey: ordersKey });
      await queryClient.cancelQueries({ queryKey: pagePrefix });

      const previousOrders = queryClient.getQueryData<FactoriesWorkOrder[]>(ordersKey);
      const factory = queryClient.getQueryData<FactoriesFactory>(factoryDetailKey(organizationId, factoryId));
      const line = factory?.lines?.find((candidate) => candidate.name === variables.lineName);
      const current = cachedWorkOrderFromLists(queryClient, organizationId, factoryId, variables.orderId);

      // Placing the card needs the line's id (and its steps, for the phase
      // label); skip the patch when the factory detail isn't cached yet.
      // Dispatch still succeeds — the card just waits for the invalidated
      // list below to move it, same as before this change.
      if (line && current) {
        applyWorkOrderToListCaches(
          queryClient,
          organizationId,
          factoryId,
          variables.orderId,
          buildOptimisticDispatchedOrder(current, line, new Date().toISOString()),
        );
      }

      return { previousOrders };
    },
    onError: (_error, _variables, context) => {
      if (context?.previousOrders) {
        queryClient.setQueryData(workOrdersKey(organizationId, factoryId), context.previousOrders);
      }
      void queryClient.invalidateQueries({ queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId) });
    },
    onSuccess: (order, variables) => {
      // Reconcile the optimistic placeholder with the real dispatch id,
      // execution id, and run refs, so the card doesn't flicker back to
      // Backlog before the invalidated queries below refetch.
      applyWorkOrderToListCaches(queryClient, organizationId, factoryId, variables.orderId, order);
      queryClient.setQueryData(workOrderDetailKey(organizationId, factoryId, variables.orderId), order);
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
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
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
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
      invalidateWorkOrderLists(queryClient, organizationId, factoryId);
      void queryClient.invalidateQueries({
        queryKey: workOrderDetailKey(organizationId, factoryId, variables.orderId),
      });
      void queryClient.invalidateQueries({
        queryKey: workOrderEventsKey(organizationId, factoryId, variables.orderId),
      });
    },
  });
}

export async function fetchFactoryAutomations(organizationId: string, factoryId: string): Promise<FactoryAutomation[]> {
  const response = await factoriesListFactoryAutomations(
    withOrganizationHeader({
      organizationId,
      path: { factoryId },
    }),
  );
  return response.data?.automations ?? [];
}

export function useFactoryAutomations(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryAppsKey(organizationId, factoryId),
    queryFn: () => fetchFactoryAutomations(organizationId, factoryId),
    enabled: Boolean(organizationId && factoryId),
  });
}

export function useCreateFactoryAutomation(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name?: string; columnKey?: string }) => {
      const response = await factoriesCreateFactoryAutomation(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            name: input.name,
            columnKey: input.columnKey,
          },
        }),
      );
      if (!response.data?.automation) {
        throw new Error("Failed to create automation");
      }
      return response.data.automation;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useDeleteFactoryAutomation(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (automationId: string) => {
      await factoriesDeleteFactoryAutomation(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, automationId },
        }),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
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

export type { FactoryAutomation, FactoriesFactoryLine, FactoryLineStep };
