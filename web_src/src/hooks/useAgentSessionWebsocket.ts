import { useCallback, useRef } from "react";
import useWebSocket from "react-use-websocket";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { agentChatKeys } from "./useAgentChats";
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
  if (data.event === "assistant_delta") {
    callbacks.onAssistantDelta?.(data.extra?.text ?? "");
    return;
  }
  if (data.event === "assistant_message" || data.event === "tool_started" || data.event === "tool_finished") {
    handlePersistedMessage(data, callbacks, queryClient);
    return;
  }
  if (data.event === "stream_started" || data.event === "turn_completed" || data.event === "session_failed") {
    callbacks.onStatusChange?.(data.status ?? "", data.error);
    // Refresh the chat list so status pills (streaming → idle) update.
    void queryClient.invalidateQueries({ queryKey: agentChatKeys.all });
  }
}

// Upsert before notifying the consumer so the persisted row is in the
// cached list before its streaming buffer is cleared.
function handlePersistedMessage(
  data: { sessionId: string; message: AgentMessage },
  callbacks: AgentStreamCallbacks,
  queryClient: QueryClient,
): void {
  if (data.sessionId && data.message) {
    upsertMessageInCache(queryClient, data.sessionId, data.message);
  }
  callbacks.onPersistedMessage?.(data.message);
}

function upsertMessageInCache(queryClient: QueryClient, sessionId: string, message: AgentMessage): void {
  queryClient.setQueryData<AgentMessage[]>(agentChatKeys.messages(sessionId), (prev) => {
    if (!prev) return prev;
    const existing = prev.findIndex((m) => m.id === message.id);
    if (existing === -1) return [...prev, message];
    const next = prev.slice();
    next[existing] = message;
    return next;
  });
}

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/agents/sessions/`;

export type AgentStreamCallbacks = {
  onAssistantDelta?: (text: string) => void;
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
