import type { AgentsAgentChatInfo, AgentsAgentChatMessage } from "@/api-client";

export type AgentChat = {
  id: string;
  canvasId: string;
  provider: string;
  status: string;
  createdAt: string | null;
  updatedAt: string | null;
};

// Image attached to a stored message. Bytes are served out-of-band by the
// image endpoint, so the message carries a URL rather than inline base64.
export type AgentMessageImage = {
  mediaType: string;
  url: string;
};

// Image being composed/sent by the client, carrying the base64 payload.
export type AgentOutgoingImage = {
  mediaType: string;
  data: string;
};

export type AgentMessage = {
  id: string;
  role: string;
  content: string;
  toolName: string;
  toolCallId: string;
  toolStatus: string;
  images?: AgentMessageImage[];
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
    }
  | {
      sessionId: string;
      event: "session_notice";
      error?: string;
    }
  | {
      sessionId: string;
      event: "outcome_evaluation_start" | "outcome_evaluation_end";
      extra?: {
        iteration?: number;
        result?: string;
        explanation?: string;
      };
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

export function fromApiMessage(
  input: AgentsAgentChatMessage | undefined,
  chatId: string,
  organizationId: string | undefined,
): AgentMessage | null {
  if (!input || !input.id) return null;
  const messageId = input.id;
  return {
    id: messageId,
    role: input.role ?? "",
    content: input.content ?? "",
    toolName: input.toolName ?? "",
    toolCallId: input.toolCallId ?? "",
    toolStatus: input.toolStatus ?? "",
    images: (input.images ?? [])
      // Index matches the image's position in the stored message, which the
      // server endpoint uses to address it; map before filtering to preserve it.
      .map((image, index) => ({
        mediaType: image.mediaType ?? "",
        url: agentMessageImageUrl(chatId, messageId, index, organizationId),
      }))
      .filter((image) => Boolean(image.mediaType)),
    createdAt: input.createdAt ?? null,
  };
}

function agentMessageImageUrl(
  chatId: string,
  messageId: string,
  index: number,
  organizationId: string | undefined,
): string {
  const query = organizationId ? `?organization_id=${encodeURIComponent(organizationId)}` : "";
  return `/api/v1/agents/chats/${chatId}/messages/${messageId}/images/${index}${query}`;
}
