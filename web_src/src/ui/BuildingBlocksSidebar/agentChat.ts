import type { Dispatch, SetStateAction } from "react";
import {
  agentsCreateAgentChat,
  agentsListAgentChatMessages,
  agentsListAgentChats,
  agentsResumeAgentChat,
} from "@/api-client";
import type {
  AgentsAgentChatInfo,
  AgentsAgentChatMessage,
  AgentsCreateAgentChatResponse,
  AgentsListAgentChatMessagesResponse,
  AgentsListAgentChatsResponse,
  AgentsResumeAgentChatResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { consumeChatResponseStream } from "./agentChatSupport";
import {
  addLocalPromptMessages,
  applyChatPromptFailure,
  applyStreamOutcome,
  clearChatPrompt,
  prependChatSession,
} from "./agentChatUi";
import type { AiCanvasOperation } from "./index";

export type AiBuilderMessage = {
  id: string;
  role: "user" | "assistant" | "tool";
  content: string;
  toolCallId?: string;
  toolStatus?: "running" | "completed";
};

export type AiBuilderProposal = {
  id: string;
  summary: string;
  operations: AiCanvasOperation[];
};

export type AiChatSession = {
  id: string;
  title: string;
  initialMessage?: string;
  createdAt?: string;
};

const AI_MAX_STORED_MESSAGES = 50;
const TEST_MODEL_SENTINEL = "success (no tool calls)";
const TEST_MODE_HINT =
  "Agent is running in test mode. Set AI_MODEL in agent/.env to a real model and configure agent credentials to get canvas-aware answers.";
const UNTITLED_CHAT_SESSION = "Untitled conversation";

export function pushAiMessages(
  previous: AiBuilderMessage[],
  next: AiBuilderMessage | AiBuilderMessage[],
): AiBuilderMessage[] {
  const nextMessages = Array.isArray(next) ? next : [next];
  const merged = [...previous, ...nextMessages];
  if (merged.length <= AI_MAX_STORED_MESSAGES) {
    return merged;
  }

  return merged.slice(-AI_MAX_STORED_MESSAGES);
}

function trimAiMessages(messages: AiBuilderMessage[]): AiBuilderMessage[] {
  if (messages.length <= AI_MAX_STORED_MESSAGES) {
    return messages;
  }

  return messages.slice(-AI_MAX_STORED_MESSAGES);
}

function insertAiMessageBefore(
  previous: AiBuilderMessage[],
  next: AiBuilderMessage,
  beforeId: string,
): AiBuilderMessage[] {
  const beforeIndex = previous.findIndex((message) => message.id === beforeId);
  if (beforeIndex < 0) {
    return pushAiMessages(previous, next);
  }

  const updated = [...previous.slice(0, beforeIndex), next, ...previous.slice(beforeIndex)];
  return trimAiMessages(updated);
}

function formatToolLabel(toolName: string): string {
  const normalized = toolName.trim().toLowerCase();
  const labelByTool: Record<string, string> = {
    get_canvas_shape: "Reading canvas structure",
    get_canvas_details: "Reading canvas details",
    list_available_blocks: "Listing available components",
  };
  if (labelByTool[normalized]) {
    return labelByTool[normalized];
  }

  const words = normalized.replace(/[_-]+/g, " ").replace(/\s+/g, " ").trim();
  if (!words) {
    return "Running tool";
  }

  return words.charAt(0).toUpperCase() + words.slice(1);
}

function parseChatIdFromUrl(url: string): string | null {
  const match = url.match(/\/agents\/chats\/([^/]+)\/stream\/?$/);
  if (!match || !match[1]) {
    return null;
  }

  return match[1];
}

function normalizePersistedMessage(message: AgentsAgentChatMessage): AiBuilderMessage | null {
  const id = typeof message.id === "string" ? message.id : "";
  const role = message.role;
  const content = typeof message.content === "string" ? message.content : "";
  const toolCallId = typeof message.toolCallId === "string" ? message.toolCallId : undefined;
  const toolStatus =
    message.toolStatus === "running" || message.toolStatus === "completed" ? message.toolStatus : undefined;

  if (!id || (role !== "user" && role !== "assistant" && role !== "tool")) {
    return null;
  }

  return {
    id,
    role,
    content,
    toolCallId,
    toolStatus,
  };
}

function normalizeChatSession(chat: AgentsAgentChatInfo): AiChatSession | null {
  const id = typeof chat.id === "string" ? chat.id.trim() : "";
  if (!id) {
    return null;
  }

  const initialMessage = typeof chat.initialMessage === "string" ? chat.initialMessage.trim() : "";
  const createdAt = typeof chat.createdAt === "string" && chat.createdAt.trim().length > 0 ? chat.createdAt : undefined;

  return {
    id,
    title: initialMessage || UNTITLED_CHAT_SESSION,
    initialMessage: initialMessage || undefined,
    createdAt,
  };
}

function normalizeChatSessions(payload: AgentsListAgentChatsResponse | undefined): AiChatSession[] {
  return (payload?.chats ?? [])
    .map((chat) => normalizeChatSession(chat))
    .filter((chat): chat is AiChatSession => Boolean(chat));
}

function normalizePersistedMessages(payload: AgentsListAgentChatMessagesResponse | undefined): AiBuilderMessage[] {
  return trimAiMessages(
    (payload?.messages ?? [])
      .map((message) => normalizePersistedMessage(message))
      .filter((message): message is AiBuilderMessage => Boolean(message)),
  );
}

function requireChatSessionPayload(payload: AgentsCreateAgentChatResponse | AgentsResumeAgentChatResponse): {
  token: string;
  url: string;
} {
  const token = typeof payload.token === "string" ? payload.token.trim() : "";
  const url = typeof payload.url === "string" ? payload.url.trim() : "";

  if (!token || !url) {
    throw new Error("Invalid chat session response");
  }

  return { token, url };
}

export async function loadChatSessions({
  canvasId,
  organizationId,
}: {
  canvasId?: string;
  organizationId?: string;
}): Promise<AiChatSession[]> {
  if (!canvasId || !organizationId) {
    return [];
  }

  const listResponse = await agentsListAgentChats(
    withOrganizationHeader({
      organizationId,
      query: {
        canvasId,
      },
    }),
  );

  return normalizeChatSessions(listResponse.data);
}

export async function loadChatConversation({
  chatId,
  canvasId,
  organizationId,
}: {
  chatId?: string | null;
  canvasId?: string;
  organizationId?: string;
}): Promise<AiBuilderMessage[]> {
  if (!canvasId || !organizationId || !chatId) {
    return [];
  }

  const messagesResponse = await agentsListAgentChatMessages(
    withOrganizationHeader({
      organizationId,
      path: {
        chatId,
      },
      query: {
        canvasId,
      },
    }),
  );

  return normalizePersistedMessages(messagesResponse.data);
}

type SendChatPromptArgs = {
  value?: string;
  aiInput: string;
  currentChatId: string | null;
  canvasId?: string;
  organizationId?: string;
  isGeneratingResponse: boolean;
  setChatSessions?: Dispatch<SetStateAction<AiChatSession[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  focusInput: () => void;
};

async function createChatSession({
  canvasId,
  createdNewChat,
  currentChatId,
  nextPrompt,
  organizationId,
  setChatSessions,
}: {
  canvasId: string;
  createdNewChat: boolean;
  currentChatId: string | null;
  nextPrompt: string;
  organizationId: string;
  setChatSessions?: Dispatch<SetStateAction<AiChatSession[]>>;
}): Promise<{ chatId: string; token: string; url: string }> {
  const sessionResponse = currentChatId
    ? await agentsResumeAgentChat(
        withOrganizationHeader({
          organizationId,
          path: {
            chatId: currentChatId,
          },
          body: {
            canvasId,
          },
        }),
      )
    : await agentsCreateAgentChat(
        withOrganizationHeader({
          organizationId,
          body: {
            canvasId,
          },
        }),
      );

  const tokenPayload = requireChatSessionPayload(sessionResponse.data);
  const chatId = currentChatId || parseChatIdFromUrl(tokenPayload.url);
  if (!chatId) {
    throw new Error("Invalid chat session response");
  }

  if (createdNewChat) {
    prependChatSession({
      chatId,
      nextPrompt,
      setChatSessions,
    });
  }

  return {
    chatId,
    token: tokenPayload.token,
    url: tokenPayload.url,
  };
}

async function fetchChatStreamResponse({
  nextPrompt,
  token,
  url,
}: {
  nextPrompt: string;
  token: string;
  url: string;
}): Promise<Response> {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "text/event-stream",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      question: nextPrompt,
    }),
  });

  if (!response.ok || !response.body) {
    const responseText = await response.text();
    throw new Error(responseText || `Request failed with status ${response.status}`);
  }

  return response;
}

export async function sendChatPrompt({
  value,
  aiInput,
  currentChatId,
  canvasId,
  organizationId,
  isGeneratingResponse,
  setChatSessions,
  setCurrentChatId,
  setAiMessages,
  setAiInput,
  setAiError,
  setIsGeneratingResponse,
  setPendingProposal,
  focusInput,
}: SendChatPromptArgs): Promise<void> {
  const nextPrompt = (value ?? aiInput).trim();
  const createdNewChat = !currentChatId;
  if (!nextPrompt || isGeneratingResponse || !canvasId || !organizationId) {
    return;
  }

  if (nextPrompt.toLowerCase() === "/clear") {
    clearChatPrompt({
      setAiMessages,
      setCurrentChatId,
      setPendingProposal,
      setAiError,
      setAiInput,
      focusInput,
    });
    return;
  }

  const assistantMessageId = `assistant-${Date.now()}`;
  addLocalPromptMessages({
    assistantMessageId,
    pushAiMessages,
    nextPrompt,
    setAiMessages,
    setAiInput,
    setAiError,
    setIsGeneratingResponse,
    focusInput,
  });
  let pendingNewChatId: string | null = null;

  try {
    setPendingProposal(null);

    const session = await createChatSession({
      canvasId,
      currentChatId,
      createdNewChat,
      nextPrompt,
      organizationId,
      setChatSessions,
    });
    if (createdNewChat) {
      pendingNewChatId = session.chatId;
    }

    const response = await fetchChatStreamResponse({
      nextPrompt,
      token: session.token,
      url: session.url,
    });

    const { assistantContentSnapshot, streamedAnyAnswer, runModel } = await consumeChatResponseStream({
      response,
      assistantMessageId,
      setAiMessages,
      setPendingProposal,
      insertAiMessageBefore,
      trimAiMessages,
      formatToolLabel,
      testModelSentinel: TEST_MODEL_SENTINEL,
      testModeHint: TEST_MODE_HINT,
    });

    applyStreamOutcome({
      assistantContentSnapshot,
      assistantMessageId,
      runModel,
      setAiMessages,
      streamedAnyAnswer,
      testModeHint: TEST_MODE_HINT,
      testModelSentinel: TEST_MODEL_SENTINEL,
    });
  } catch (error) {
    applyChatPromptFailure({
      assistantMessageId,
      error,
      pushAiMessages,
      setAiError,
      setAiMessages,
      trimAiMessages,
    });
  } finally {
    setIsGeneratingResponse(false);
    if (pendingNewChatId) {
      setCurrentChatId(pendingNewChatId);
    }
  }
}
