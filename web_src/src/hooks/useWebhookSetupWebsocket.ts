import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { useWebSocket } from "@/lib/reactUseWebsocket";

import { integrationKeys } from "./useIntegrations";

export function WebhookSetupListener({ organizationId }: { organizationId: string }) {
  const { canAct } = usePermissions();
  const queryClient = useQueryClient();
  const enabled = Boolean(organizationId) && (canAct("integrations", "read") || canAct("factories", "read"));

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: integrationKeys.connected(organizationId),
      predicate: ({ queryKey }) =>
        queryKey[3] === "workspace" || queryKey.length <= integrationKeys.integration(organizationId, "").length,
    });
    void queryClient.invalidateQueries({
      queryKey: ["factories", organizationId],
      predicate: ({ queryKey }) => queryKey.length === 4 && queryKey[3] === "intakes",
    });
  }, [organizationId, queryClient]);

  const onMessage = useCallback(
    (message: MessageEvent<string>) => {
      try {
        if (JSON.parse(message.data)?.event === "webhook_setup_changed") refresh();
      } catch {
        // Ignore unrecognized messages on the live update connection.
      }
    },
    [refresh],
  );

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const url = `${protocol}//${window.location.host}/ws/integrations/webhook-setup?organization_id=${encodeURIComponent(organizationId)}`;
  useWebSocket(
    enabled ? url : null,
    {
      onOpen: refresh,
      onMessage,
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      reconnectInterval: 3000,
      heartbeat: false,
    },
    enabled,
  );
  return null;
}
