import { useCallback, useRef } from "react";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { useQueryClient } from "@tanstack/react-query";
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

export function invalidateFactoryWorkOrderQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
  orderId?: string,
): void {
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.workOrders(organizationId, factoryId),
  });

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
