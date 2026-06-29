import type { AgentsAgentChatImageMediaType, AgentsAgentChatInfo, AgentsAgentChatMessage } from "@/api-client";

export type AgentImageMediaType = AgentsAgentChatImageMediaType;

export type AgentChat = {
  id: string;
  canvasId: string;
  provider: string;
  status: string;
  createdAt: string | null;
  updatedAt: string | null;
};

export type AgentMessageImage = {
  mediaType: string;
  url: string;
};

export type AgentOutgoingImage = {
  mediaType: AgentImageMediaType;
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
      .map((image, index) => ({
        mediaType: apiImageMediaTypeToMime(image.mediaType),
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

export function apiImageMediaTypeToMime(mediaType: string | undefined): string {
  switch (mediaType) {
    case "MEDIA_TYPE_PNG":
      return "image/png";
    case "MEDIA_TYPE_JPEG":
      return "image/jpeg";
    case "MEDIA_TYPE_GIF":
      return "image/gif";
    case "MEDIA_TYPE_WEBP":
      return "image/webp";
    default:
      return "";
  }
}

export function mimeToApiImageMediaType(mediaType: string): AgentImageMediaType {
  switch (mediaType) {
    case "image/png":
      return "MEDIA_TYPE_PNG";
    case "image/jpeg":
      return "MEDIA_TYPE_JPEG";
    case "image/gif":
      return "MEDIA_TYPE_GIF";
    case "image/webp":
      return "MEDIA_TYPE_WEBP";
    default:
      return "MEDIA_TYPE_UNSPECIFIED";
  }
}
