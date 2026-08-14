import { useCallback, useRef } from "react";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { useQueryClient } from "@tanstack/react-query";
import { factoryQueryKeys } from "./useFactoryData";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/factories/`;

type FactoryWebsocketPayload = {
  factoryId?: string;
  orderId?: string;
  appId?: string;
  reason?: string;
};

type FactoryWebsocketMessage = {
  event?: string;
  payload?: FactoryWebsocketPayload;
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
}

export function invalidateFactoryAppQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
): void {
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.apps(organizationId, factoryId),
  });
}

export function invalidateFactoryDetailQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
): void {
  void queryClient.invalidateQueries({
    queryKey: factoryQueryKeys.detail(organizationId, factoryId),
  });
}

export function useFactoryWebsocket(organizationId: string, factoryId: string, enabled = true): void {
  const queryClient = useQueryClient();
  const hasConnectedOnce = useRef(false);

  const invalidateWorkOrders = useCallback(
    (orderId?: string) => {
      if (!organizationId || !factoryId) {
        return;
      }
      invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
    },
    [queryClient, organizationId, factoryId],
  );

  const invalidateApps = useCallback(() => {
    if (!organizationId || !factoryId) {
      return;
    }
    invalidateFactoryAppQueries(queryClient, organizationId, factoryId);
  }, [queryClient, organizationId, factoryId]);

  const invalidateDetail = useCallback(() => {
    if (!organizationId || !factoryId) {
      return;
    }
    invalidateFactoryDetailQueries(queryClient, organizationId, factoryId);
  }, [queryClient, organizationId, factoryId]);

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseFactoryEvent(event);
      if (!data?.event) {
        return;
      }
      if (data.payload?.factoryId && data.payload.factoryId !== factoryId) {
        return;
      }

      switch (data.event) {
        case "work_order_updated":
          invalidateWorkOrders(data.payload?.orderId);
          return;
        case "factory_app_updated":
          invalidateApps();
          return;
        case "factory_updated":
          invalidateDetail();
          return;
        default:
          return;
      }
    },
    [factoryId, invalidateApps, invalidateDetail, invalidateWorkOrders],
  );

  const onOpen = useCallback(() => {
    if (!hasConnectedOnce.current) {
      hasConnectedOnce.current = true;
      return;
    }

    // Catch updates missed while disconnected; WS is the only push channel.
    invalidateWorkOrders();
    invalidateApps();
    invalidateDetail();
  }, [invalidateApps, invalidateDetail, invalidateWorkOrders]);

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
