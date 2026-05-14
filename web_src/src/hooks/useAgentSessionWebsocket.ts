import { useCallback, useRef } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { upsertAgentMessageInCache } from "./useAgentChats";
import type { AgentMessage, AgentSessionWebsocketEvent } from "@/components/AgentSidebar/types";

function parseAgentEvent(event: MessageEvent<unknown>): AgentSessionWebsocketEvent | null {
  try {
    return JSON.parse(event.data as string) as AgentSessionWebsocketEvent;
  } catch (error) {
    console.warn("agent session ws: failed to parse message", error);
    return null;
  }
}

function dispatchAgentEvent(
  data: AgentSessionWebsocketEvent,
  callbacks: AgentStreamCallbacks,
  queryClient: QueryClient,
): void {
  if (data.event === "assistant_message" || data.event === "tool_started" || data.event === "tool_finished") {
    handlePersistedMessage(data, callbacks, queryClient);
    return;
  }
  if (data.event === "stream_started" || data.event === "turn_completed" || data.event === "session_failed") {
    callbacks.onStatusChange?.(data.status ?? "", data.error);
  }
}

function handlePersistedMessage(
  data: { sessionId: string; message: AgentMessage },
  callbacks: AgentStreamCallbacks,
  queryClient: QueryClient,
): void {
  if (data.sessionId && data.message) {
    upsertAgentMessageInCache(queryClient, data.sessionId, data.message);
  }
  callbacks.onPersistedMessage?.(data.message);
}

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/agents/sessions/`;

export type AgentStreamCallbacks = {
  onPersistedMessage?: (message: AgentMessage) => void;
  onStatusChange?: (status: string, error?: string) => void;
};

export function useAgentSessionWebsocket(
  sessionId: string | null,
  organizationId: string | undefined,
  callbacks: AgentStreamCallbacks,
  enabled = true,
): void {
  const queryClient = useQueryClient();
  const callbacksRef = useRef(callbacks);
  callbacksRef.current = callbacks;

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseAgentEvent(event);
      if (!data) return;
      dispatchAgentEvent(data, callbacksRef.current, queryClient);
    },
    [queryClient],
  );

  const url = sessionId && organizationId ? `${SOCKET_SERVER_URL}${sessionId}?organization_id=${organizationId}` : null;
  useWebSocket(
    url,
    {
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      reconnectInterval: 3000,
      heartbeat: false,
      share: false,
      onMessage,
    },
    enabled && url !== null,
  );
}
