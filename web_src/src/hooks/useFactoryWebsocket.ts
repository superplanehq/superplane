import type { FactoriesWorkOrder, FactoriesWorkOrderCheckScore, FactoriesWorkOrderSummary } from "@/api-client";
import { factoriesDescribeWorkOrder } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef } from "react";
import { canvasKeys } from "./useCanvasData";
import { factoryQueryKeys } from "./useFactoryData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/factories/`;

type FactoryWorkOrderUpdatedPayload = {
  factoryId?: string;
  orderId?: string;
  reason?: string;
};

type FactoryWebsocketMessage = {
  event?: string;
  payload?: FactoryWorkOrderUpdatedPayload;
};

function parseFactoryEvent(event: MessageEvent<unknown>): FactoryWebsocketMessage | null {
  try {
    return JSON.parse(event.data as string) as FactoryWebsocketMessage;
  } catch (error) {
    console.warn("factory ws: failed to parse message", error);
    return null;
  }
}

export function canvasRunsForWorkOrders(orders: FactoriesWorkOrder[]): Array<{ appId: string; runId: string }> {
  const seen = new Set<string>();
  const runs: Array<{ appId: string; runId: string }> = [];
  for (const order of orders) {
    for (const dispatch of order.lineDispatches ?? []) {
      for (const execution of dispatch.stepExecutions ?? []) {
        const appId = execution.run?.appId;
        const runId = execution.run?.id;
        if (!appId || !runId) {
          continue;
        }
        const key = `${appId}:${runId}`;
        if (seen.has(key)) {
          continue;
        }
        seen.add(key);
        runs.push({ appId, runId });
      }
    }
  }
  return runs;
}

type WorkOrderQueryClient = ReturnType<typeof useQueryClient>;

function cachedWorkOrdersForInvalidation(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): FactoriesWorkOrder[] {
  const list =
    queryClient.getQueryData<FactoriesWorkOrder[]>(factoryQueryKeys.workOrders(organizationId, factoryId)) ?? [];
  if (!orderId) {
    return list;
  }

  const detail = queryClient.getQueryData<FactoriesWorkOrder>(
    factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
  );
  return [...list.filter((order) => order.id === orderId), ...(detail && detail.id === orderId ? [detail] : [])];
}

function invalidateOrdersList(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string) {
  // exact: the detail, events, and artifacts keys extend this prefix.
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrders(organizationId, factoryId),
    exact: true,
  });
}

function invalidatePullRequests(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string) {
  void queryClient.invalidateQueries({
    queryKey: ["factories", organizationId, factoryId, "pull-requests"],
  });
}

function invalidateBacklogAnalysisRuns(queryClient: WorkOrderQueryClient, organizationId: string) {
  // A draft created through the API has no optimistic "analyzing" flag.
  // This refetch picks up the Backlog run once the factory creates it.
  void queryClient.invalidateQueries({
    queryKey: ["backlog-analysis-runs", organizationId],
  });
}

function invalidateCachedCanvasRuns(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId?: string,
) {
  for (const run of canvasRunsForWorkOrders(
    cachedWorkOrdersForInvalidation(queryClient, organizationId, factoryId, orderId),
  )) {
    void queryClient.invalidateQueries({
      queryKey: canvasKeys.run(run.appId, run.runId),
    });
  }
}

function checkScoresFromChecks(order: FactoriesWorkOrder): FactoriesWorkOrderCheckScore[] {
  return (order.checks ?? []).map((check) => ({
    key: check.key,
    name: check.name,
    score: check.score,
    maxScore: check.maxScore,
  }));
}

const SUMMARY_FIELDS = [
  "id",
  "title",
  "description",
  "state",
  "result",
  "createdAt",
  "updatedAt",
  "assignees",
  "createdBy",
  "totalTokens",
  "totalCostCents",
  "number",
  "key",
  "lineDispatches",
  "statusNotes",
  "origin",
  "totalDurationSeconds",
] as const satisfies ReadonlyArray<keyof FactoriesWorkOrder & keyof FactoriesWorkOrderSummary>;

function summaryFromDescribedOrder(order: FactoriesWorkOrder): FactoriesWorkOrderSummary {
  const summary: FactoriesWorkOrderSummary = { checkScores: checkScoresFromChecks(order) };
  for (const field of SUMMARY_FIELDS) {
    if (order[field] !== undefined) {
      summary[field] = order[field] as never;
    }
  }
  return summary;
}

function patchedWorkOrder(order: FactoriesWorkOrderSummary, described: FactoriesWorkOrder): FactoriesWorkOrderSummary {
  return { ...order, ...summaryFromDescribedOrder(described), planningSession: described.planningSession };
}

function patchCachedWorkOrder(
  orders: FactoriesWorkOrderSummary[] | undefined,
  orderId: string,
  described: FactoriesWorkOrder,
): FactoriesWorkOrderSummary[] | undefined {
  if (!orders) {
    return orders;
  }
  const index = orders.findIndex((order) => order.id === orderId);
  if (index < 0) {
    return [patchedWorkOrder({ id: described.id ?? orderId }, described), ...orders];
  }
  return orders.map((order) => (order.id === orderId ? patchedWorkOrder(order, described) : order));
}

async function describeWorkOrder(
  organizationId: string,
  factoryId: string,
  orderId: string,
): Promise<FactoriesWorkOrder> {
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
}

function cachedWorkOrderIds(queryClient: WorkOrderQueryClient, organizationId: string, factoryId: string): string[] {
  const orders =
    queryClient.getQueryData<FactoriesWorkOrderSummary[]>(factoryQueryKeys.workOrders(organizationId, factoryId)) ?? [];
  return orders.flatMap((order) => (order.id ? [order.id] : []));
}

function invalidateTaskActivity(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderIds: string[],
) {
  invalidatePullRequests(queryClient, organizationId, factoryId);
  invalidateBacklogAnalysisRuns(queryClient, organizationId);
  for (const orderId of orderIds) {
    invalidateCachedCanvasRuns(queryClient, organizationId, factoryId, orderId);
    void queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
    });
    void queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId),
    });
  }
}

async function refreshUpdatedWorkOrder(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
  isCurrent: () => boolean,
) {
  invalidateTaskActivity(queryClient, organizationId, factoryId, [orderId]);
  const order = await describeWorkOrder(organizationId, factoryId, orderId);
  if (!isCurrent()) {
    return;
  }
  queryClient.setQueryData(factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId), order);
  queryClient.setQueryData<FactoriesWorkOrderSummary[]>(
    factoryQueryKeys.workOrders(organizationId, factoryId),
    (orders) => patchCachedWorkOrder(orders, orderId, order),
  );
}

function invalidateFactoryWorkOrdersOnReconnect(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
) {
  invalidateOrdersList(queryClient, organizationId, factoryId);
  invalidateTaskActivity(
    queryClient,
    organizationId,
    factoryId,
    cachedWorkOrderIds(queryClient, organizationId, factoryId),
  );
}

export function invalidateFactoryWorkOrderQueries(
  queryClient: WorkOrderQueryClient,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): void {
  invalidateOrdersList(queryClient, organizationId, factoryId);
  invalidatePullRequests(queryClient, organizationId, factoryId);
  invalidateBacklogAnalysisRuns(queryClient, organizationId);
  invalidateCachedCanvasRuns(queryClient, organizationId, factoryId, orderId);

  if (!orderId) {
    return;
  }

  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
    exact: true,
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId),
  });
}

export function useFactoryWebsocket(organizationId: string, factoryId: string, enabled = true): void {
  const queryClient = useQueryClient();
  const hasConnectedOnce = useRef(false);
  const refreshVersion = useRef(new Map<string, number>());

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseFactoryEvent(event);
      if (!data || data.event !== "work_order_updated") {
        return;
      }
      if (!organizationId || !factoryId) {
        return;
      }
      const orderId = data.payload?.orderId;
      if (!orderId || (data.payload?.factoryId && data.payload.factoryId !== factoryId)) {
        return;
      }
      const version = (refreshVersion.current.get(orderId) ?? 0) + 1;
      refreshVersion.current.set(orderId, version);
      void refreshUpdatedWorkOrder(
        queryClient,
        organizationId,
        factoryId,
        orderId,
        () => refreshVersion.current.get(orderId) === version,
      ).catch((error) => {
        if (refreshVersion.current.get(orderId) !== version) {
          return;
        }
        const message = getApiErrorMessage(error, "");
        if (message) {
          console.warn("factory ws: failed to refresh work order", message);
        }
        invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
      });
    },
    [queryClient, organizationId, factoryId],
  );

  const onOpen = useCallback(() => {
    if (!hasConnectedOnce.current) {
      hasConnectedOnce.current = true;
      return;
    }
    if (!organizationId || !factoryId) {
      return;
    }

    // Catch updates missed while disconnected; WS is the only push channel.
    invalidateFactoryWorkOrdersOnReconnect(queryClient, organizationId, factoryId);
  }, [queryClient, organizationId, factoryId]);

  const url = organizationId && factoryId ? `${SOCKET_SERVER_URL}${factoryId}?organization_id=${organizationId}` : null;

  useWebSocket(
    url,
    {
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      reconnectInterval: 3000,
      heartbeat: false,
      share: false,
      onMessage,
      onOpen,
    },
    enabled && url !== null,
  );
}
