import type { AgentsAgentChatInfo, AgentsAgentChatMessage } from "@/api-client";

export type AgentChat = {
  id: string;
  canvasId: string;
  provider: string;
  status: string;
  createdAt: string | null;
  updatedAt: string | null;
};

export type AgentMessage = {
  id: string;
  role: string;
  content: string;
  toolName: string;
  toolCallId: string;
  toolStatus: string;
  createdAt: string | null;
};

export type AgentSessionWebsocketEvent =
  | {
      sessionId: string;
      event: "assistant_message" | "tool_started" | "tool_finished";
      messageId: string;
      message: AgentMessage;
    }
  | {
      sessionId: string;
      event: "stream_started" | "turn_completed" | "session_failed";
      status?: string;
      error?: string;
    };

export function fromApiChat(input: AgentsAgentChatInfo | undefined): AgentChat | null {
  if (!input || !input.id) return null;
  return {
    id: input.id,
    canvasId: input.canvasId ?? "",
    provider: input.provider ?? "",
    status: input.status ?? "idle",
    createdAt: input.createdAt ?? null,
    updatedAt: input.updatedAt ?? null,
  };
}

export function fromApiMessage(input: AgentsAgentChatMessage | undefined): AgentMessage | null {
  if (!input || !input.id) return null;
  return {
    id: input.id,
    role: input.role ?? "",
    content: input.content ?? "",
    toolName: input.toolName ?? "",
    toolCallId: input.toolCallId ?? "",
    toolStatus: input.toolStatus ?? "",
    createdAt: input.createdAt ?? null,
  };
}
