import type { FactoriesWorkOrder } from "@/api-client";
import { useWebSocket } from "@/lib/reactUseWebsocket";
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

export function invalidateFactoryWorkOrderQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): void {
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrders(organizationId, factoryId),
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.detail(organizationId, factoryId),
  });

  for (const run of canvasRunsForWorkOrders(
    cachedWorkOrdersForInvalidation(queryClient, organizationId, factoryId, orderId),
  )) {
    void queryClient.invalidateQueries({
      queryKey: canvasKeys.run(run.appId, run.runId),
    });
  }

  if (!orderId) {
    return;
  }

  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId),
  });
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
  });
  // Artifact state (e.g. a PR flipping open/draft/closed/merged via
  // updateWorkOrderArtifact) and new artifacts (addWorkOrderArtifact)
  // both land here — without this, the sidebar artifact list and the
  // timeline's live-data overlay silently go stale until a manual reload.
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, orderId),
  });
  // Check reports (reportWorkOrderCheck) update the scorecards in place;
  // refetch so a re-scored check shows its new value and trend delta.
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrderChecks(organizationId, factoryId, orderId),
  });
}

export function useFactoryWebsocket(organizationId: string, factoryId: string, enabled = true): void {
  const queryClient = useQueryClient();
  const hasConnectedOnce = useRef(false);

  const invalidate = useCallback(
    (orderId?: string) => {
      if (!organizationId || !factoryId) {
        return;
      }
      invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
    },
    [queryClient, organizationId, factoryId],
  );

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseFactoryEvent(event);
      if (!data || data.event !== "work_order_updated") {
        return;
      }
      if (data.payload?.factoryId && data.payload.factoryId !== factoryId) {
        return;
      }
      invalidate(data.payload?.orderId);
    },
    [factoryId, invalidate],
  );

  const onOpen = useCallback(() => {
    if (!hasConnectedOnce.current) {
      hasConnectedOnce.current = true;
      return;
    }

    // Catch updates missed while disconnected; WS is the only push channel.
    invalidate();
  }, [invalidate]);

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
