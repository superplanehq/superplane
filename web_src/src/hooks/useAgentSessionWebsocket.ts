import { useCallback, useRef } from "react";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { upsertAgentMessageInCache } from "./useAgentChats";
import type { AgentMessage, AgentSessionWebsocketEvent } from "@/components/CanvasToolSidebar/types";

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
  switch (data.event) {
    case "assistant_message":
    case "tool_started":
    case "tool_finished":
      handlePersistedMessage(data, callbacks, queryClient);
      return;
    case "stream_started":
    case "turn_completed":
    case "session_failed":
      handleStatusEvent(data, callbacks);
      return;
    case "session_notice":
      callbacks.onNotice?.(data.error ?? "");
      return;
    case "outcome_evaluation_start":
    case "outcome_evaluation_end":
      handleOutcomeEvent(data, callbacks);
      return;
  }
}

function outcomePhase(event: "outcome_evaluation_start" | "outcome_evaluation_end"): "start" | "end" {
  return event === "outcome_evaluation_start" ? "start" : "end";
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

function handleStatusEvent(data: { status?: string; error?: string }, callbacks: AgentStreamCallbacks): void {
  callbacks.onStatusChange?.(data.status ?? "", data.error);
}

function handleOutcomeEvent(
  data: {
    event: "outcome_evaluation_start" | "outcome_evaluation_end";
    extra?: { iteration?: number; result?: string; explanation?: string };
  },
  callbacks: AgentStreamCallbacks,
): void {
  callbacks.onOutcomeEvent?.(outcomePhase(data.event), {
    iteration: data.extra?.iteration ?? 0,
    result: data.extra?.result,
    explanation: data.extra?.explanation,
  });
}

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/agents/sessions/`;

export type OutcomeEvaluation = {
  iteration: number;
  result?: string;
  explanation?: string;
};

export type AgentStreamCallbacks = {
  onPersistedMessage?: (message: AgentMessage) => void;
  onStatusChange?: (status: string, error?: string) => void;
  onOutcomeEvent?: (event: "start" | "end", evaluation: OutcomeEvaluation) => void;
  /** A recoverable, non-terminal provider notice (session.error). */
  onNotice?: (message: string) => void;
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
