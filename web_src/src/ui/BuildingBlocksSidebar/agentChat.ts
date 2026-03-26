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

export type AiAgentSession = {
  id: string;
  title: string;
  initialMessage?: string;
  createdAt?: string;
};

const AI_MAX_STORED_MESSAGES = 50;
const TEST_MODEL_SENTINEL = "success (no tool calls)";
const TEST_MODE_HINT =
  "Agent is running in test mode. Set AI_MODEL in agent/.env to a real model and configure agent credentials to get canvas-aware answers.";
const GENERIC_FAILURE_MESSAGE = "I couldn't generate changes right now. Please try again.";
const UNTITLED_AGENT_SESSION = "Untitled conversation";

type JsonObject = Record<string, unknown>;

type AgentChatStreamEvent =
  | { type: "run_started"; model?: string }
  | { type: "model_delta"; content?: string }
  | { type: "tool_started"; tool_name?: string; tool_call_id?: string }
  | { type: "tool_finished"; tool_name?: string; tool_call_id?: string; elapsed_ms?: number }
  | { type: "final_answer"; output?: unknown }
  | { type: "run_failed"; error?: string }
  | { type: "run_completed" }
  | { type: "done" }
  | { type: "raw_data"; content: string };

function isRecord(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeNodeRef(
  value: unknown,
): { nodeKey?: string; nodeId?: string; nodeName?: string; handleId?: string | null } | null {
  if (!isRecord(value)) {
    return null;
  }

  const nodeKey = typeof value.nodeKey === "string" ? value.nodeKey : undefined;
  const nodeId = typeof value.nodeId === "string" ? value.nodeId : undefined;
  const nodeName = typeof value.nodeName === "string" ? value.nodeName : undefined;
  const handleId = typeof value.handleId === "string" ? value.handleId : value.handleId === null ? null : undefined;

  if (!nodeKey && !nodeId && !nodeName) {
    return null;
  }

  return { nodeKey, nodeId, nodeName, handleId };
}

function normalizeAiOperation(value: unknown): AiCanvasOperation | null {
  if (!isRecord(value) || typeof value.type !== "string") {
    return null;
  }

  if (value.type === "add_node") {
    const blockName = typeof value.blockName === "string" ? value.blockName : "";
    if (!blockName) {
      return null;
    }

    const operation: AiCanvasOperation = {
      type: "add_node",
      blockName,
      nodeKey: typeof value.nodeKey === "string" ? value.nodeKey : undefined,
      nodeName: typeof value.nodeName === "string" ? value.nodeName : undefined,
    };
    if (isRecord(value.configuration)) {
      operation.configuration = value.configuration;
    }
    if (isRecord(value.position) && typeof value.position.x === "number" && typeof value.position.y === "number") {
      operation.position = { x: value.position.x, y: value.position.y };
    }
    const source = normalizeNodeRef(value.source);
    if (source) {
      operation.source = source;
    }
    return operation;
  }

  if (value.type === "connect_nodes" || value.type === "disconnect_nodes") {
    const source = normalizeNodeRef(value.source);
    const target = normalizeNodeRef(value.target);
    if (!source || !target) {
      return null;
    }

    return {
      type: value.type,
      source,
      target,
    };
  }

  if (value.type === "update_node_config") {
    const target = normalizeNodeRef(value.target);
    if (!target) {
      return null;
    }

    const operation: AiCanvasOperation = {
      type: "update_node_config",
      target,
      configuration: isRecord(value.configuration) ? value.configuration : {},
      nodeName: typeof value.nodeName === "string" ? value.nodeName : undefined,
    };
    return operation;
  }

  if (value.type === "delete_node") {
    const target = normalizeNodeRef(value.target);
    if (!target) {
      return null;
    }
    return {
      type: "delete_node",
      target,
    };
  }

  return null;
}

function normalizeAiProposal(value: unknown): AiBuilderProposal | null {
  if (!isRecord(value)) {
    return null;
  }

  const summary = typeof value.summary === "string" ? value.summary.trim() : "";
  if (!summary) {
    return null;
  }

  const operationsRaw = Array.isArray(value.operations) ? value.operations : [];
  const operations = operationsRaw
    .map((operation) => normalizeAiOperation(operation))
    .filter((operation): operation is AiCanvasOperation => Boolean(operation));
  if (operations.length === 0) {
    return null;
  }

  return {
    id: `proposal-${Date.now()}`,
    summary,
    operations,
  };
}

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

function parseSseChunk(rawChunk: string): AgentChatStreamEvent[] {
  const chunks = rawChunk.split("\n\n");
  const events: AgentChatStreamEvent[] = [];

  for (const chunk of chunks) {
    const lines = chunk.split("\n");
    const dataLines: string[] = [];
    for (const line of lines) {
      if (line.startsWith("data:")) {
        dataLines.push(line.replace(/^data:\s*/, ""));
      }
    }

    if (!dataLines.length) {
      continue;
    }

    const merged = dataLines.join("\n").trim();
    if (!merged) {
      continue;
    }

    try {
      const parsed = JSON.parse(merged);
      const normalized = normalizeStreamEvent(parsed);
      if (normalized) {
        events.push(normalized);
      }
    } catch {
      events.push({ type: "raw_data", content: merged });
    }
  }

  return events;
}

function parseAgentChatIdFromUrl(url: string): string | null {
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

function normalizeAgentSession(agent: AgentsAgentChatInfo): AiAgentSession | null {
  const id = typeof agent.id === "string" ? agent.id.trim() : "";
  if (!id) {
    return null;
  }

  const initialMessage = typeof agent.initialMessage === "string" ? agent.initialMessage.trim() : "";
  const createdAt =
    typeof agent.createdAt === "string" && agent.createdAt.trim().length > 0 ? agent.createdAt : undefined;

  return {
    id,
    title: initialMessage || UNTITLED_AGENT_SESSION,
    initialMessage: initialMessage || undefined,
    createdAt,
  };
}

function normalizeAgentSessions(payload: AgentsListAgentChatsResponse | undefined): AiAgentSession[] {
  return (payload?.chats ?? [])
    .map((agent) => normalizeAgentSession(agent))
    .filter((agent): agent is AiAgentSession => Boolean(agent));
}

function normalizePersistedMessages(payload: AgentsListAgentChatMessagesResponse | undefined): AiBuilderMessage[] {
  return trimAiMessages(
    (payload?.messages ?? [])
      .map((message) => normalizePersistedMessage(message))
      .filter((message): message is AiBuilderMessage => Boolean(message)),
  );
}

function normalizeStreamEvent(value: unknown): AgentChatStreamEvent | null {
  if (!isRecord(value) || typeof value.type !== "string") {
    return null;
  }

  switch (value.type) {
    case "run_started":
      return {
        type: "run_started",
        model: typeof value.model === "string" ? value.model : undefined,
      };
    case "model_delta":
      return {
        type: "model_delta",
        content: typeof value.content === "string" ? value.content : undefined,
      };
    case "tool_started":
      return {
        type: "tool_started",
        tool_name: typeof value.tool_name === "string" ? value.tool_name : undefined,
        tool_call_id: typeof value.tool_call_id === "string" ? value.tool_call_id : undefined,
      };
    case "tool_finished":
      return {
        type: "tool_finished",
        tool_name: typeof value.tool_name === "string" ? value.tool_name : undefined,
        tool_call_id: typeof value.tool_call_id === "string" ? value.tool_call_id : undefined,
        elapsed_ms: typeof value.elapsed_ms === "number" ? value.elapsed_ms : undefined,
      };
    case "final_answer":
      return {
        type: "final_answer",
        output: value.output,
      };
    case "run_failed":
      return {
        type: "run_failed",
        error: typeof value.error === "string" ? value.error : undefined,
      };
    case "run_completed":
      return { type: "run_completed" };
    case "done":
      return { type: "done" };
    default:
      return null;
  }
}

function requireAgentSessionPayload(payload: AgentsCreateAgentChatResponse | AgentsResumeAgentChatResponse): {
  token: string;
  url: string;
} {
  const token = typeof payload.token === "string" ? payload.token.trim() : "";
  const url = typeof payload.url === "string" ? payload.url.trim() : "";

  if (!token || !url) {
    throw new Error("Invalid agent session response");
  }

  return { token, url };
}

export async function loadAgentSessions({
  canvasId,
  organizationId,
}: {
  canvasId?: string;
  organizationId?: string;
}): Promise<AiAgentSession[]> {
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

  return normalizeAgentSessions(listResponse.data);
}

export async function loadAgentConversation({
  agentId,
  canvasId,
  organizationId,
}: {
  agentId?: string | null;
  canvasId?: string;
  organizationId?: string;
}): Promise<AiBuilderMessage[]> {
  if (!canvasId || !organizationId || !agentId) {
    return [];
  }

  const messagesResponse = await agentsListAgentChatMessages(
    withOrganizationHeader({
      organizationId,
      path: {
        chatId: agentId,
      },
      query: {
        canvasId,
      },
    }),
  );

  return normalizePersistedMessages(messagesResponse.data);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

type SendAgentChatPromptArgs = {
  value?: string;
  aiInput: string;
  currentAgentId: string | null;
  canvasId?: string;
  organizationId?: string;
  isGeneratingResponse: boolean;
  setCurrentAgentId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  focusInput: () => void;
};

export async function sendAgentChatPrompt({
  value,
  aiInput,
  currentAgentId,
  canvasId,
  organizationId,
  isGeneratingResponse,
  setCurrentAgentId,
  setAiMessages,
  setAiInput,
  setAiError,
  setIsGeneratingResponse,
  setPendingProposal,
  focusInput,
}: SendAgentChatPromptArgs): Promise<void> {
  const nextPrompt = (value ?? aiInput).trim();
  if (!nextPrompt || isGeneratingResponse || !canvasId || !organizationId) {
    return;
  }

  if (nextPrompt.toLowerCase() === "/clear") {
    setAiMessages([]);
    setCurrentAgentId(null);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
    requestAnimationFrame(() => {
      focusInput();
    });
    return;
  }

  const userMessage: AiBuilderMessage = {
    id: `user-${Date.now()}`,
    role: "user",
    content: nextPrompt,
  };
  setAiMessages((prev) => pushAiMessages(prev, userMessage));
  setAiInput("");
  requestAnimationFrame(() => {
    focusInput();
  });
  setAiError(null);
  setIsGeneratingResponse(true);
  const assistantMessageId = `assistant-${Date.now()}`;

  try {
    setAiMessages((prev) =>
      pushAiMessages(prev, {
        id: assistantMessageId,
        role: "assistant",
        content: "",
      }),
    );
    setPendingProposal(null);

    const sessionResponse = currentAgentId
      ? await agentsResumeAgentChat(
          withOrganizationHeader({
            organizationId,
            path: {
              chatId: currentAgentId,
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

    const tokenPayload = requireAgentSessionPayload(sessionResponse.data);

    const resolvedAgentId = currentAgentId || parseAgentChatIdFromUrl(tokenPayload.url);
    if (!resolvedAgentId) {
      throw new Error("Invalid agent session response");
    }
    setCurrentAgentId(resolvedAgentId);

    const response = await fetch(tokenPayload.url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
        Authorization: `Bearer ${tokenPayload.token}`,
      },
      body: JSON.stringify({
        question: nextPrompt,
      }),
    });

    if (!response.ok || !response.body) {
      const responseText = await response.text();
      throw new Error(responseText || `Request failed with status ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let streamedAnyAnswer = false;
    let assistantContentSnapshot = "";
    let runModel = "";
    let pendingRenderBuffer = "";
    let isRenderLoopRunning = false;

    const flushPendingRenderBuffer = async () => {
      if (isRenderLoopRunning) {
        return;
      }

      isRenderLoopRunning = true;
      try {
        while (pendingRenderBuffer.length > 0) {
          const nextChunk = pendingRenderBuffer.slice(0, 5);
          pendingRenderBuffer = pendingRenderBuffer.slice(5);
          assistantContentSnapshot += nextChunk;
          streamedAnyAnswer = true;
          setAiMessages((prev) =>
            prev.map((message) =>
              message.id === assistantMessageId ? { ...message, content: `${message.content}${nextChunk}` } : message,
            ),
          );
          await sleep(8);
        }
      } finally {
        isRenderLoopRunning = false;
      }
    };

    const waitForRenderLoopIdle = async () => {
      while (isRenderLoopRunning || pendingRenderBuffer.length > 0) {
        await sleep(10);
      }
    };

    const appendAssistantContent = (chunk: string) => {
      if (!chunk) return;
      pendingRenderBuffer += chunk;
      void flushPendingRenderBuffer();
    };

    const upsertToolMessage = (toolCallId: string, updater: (existing?: AiBuilderMessage) => AiBuilderMessage) => {
      setAiMessages((prev) => {
        const existingIndex = prev.findIndex((message) => message.role === "tool" && message.toolCallId === toolCallId);
        if (existingIndex >= 0) {
          const updated = [...prev];
          updated[existingIndex] = updater(updated[existingIndex]);
          return trimAiMessages(updated);
        }

        const nextMessage = updater(undefined);
        return insertAiMessageBefore(prev, nextMessage, assistantMessageId);
      });
    };

    const replaceAssistantContent = (content: string) => {
      assistantContentSnapshot = content;
      streamedAnyAnswer = true;
      setAiMessages((prev) =>
        prev.map((message) => (message.id === assistantMessageId ? { ...message, content } : message)),
      );
    };

    const processEvent = async (event: AgentChatStreamEvent) => {
      if (event.type === "run_started" && typeof event.model === "string") {
        runModel = event.model.trim().toLowerCase();
        return;
      }

      if (event.type === "model_delta" && typeof event.content === "string") {
        appendAssistantContent(event.content);
        return;
      }

      if (event.type === "tool_started") {
        const toolName = typeof event.tool_name === "string" ? event.tool_name : "unknown";
        const toolCallId =
          typeof event.tool_call_id === "string" && event.tool_call_id.trim().length > 0
            ? event.tool_call_id
            : `${toolName}-${Date.now()}`;
        const toolLabel = formatToolLabel(toolName);
        upsertToolMessage(toolCallId, (existing) => ({
          id: existing?.id || `tool-${toolCallId}`,
          role: "tool",
          content: `${toolLabel}...`,
          toolCallId,
          toolStatus: "running",
        }));
        return;
      }

      if (event.type === "tool_finished") {
        const toolName = typeof event.tool_name === "string" ? event.tool_name : "unknown";
        const toolCallId =
          typeof event.tool_call_id === "string" && event.tool_call_id.trim().length > 0
            ? event.tool_call_id
            : `${toolName}-${Date.now()}`;
        const toolLabel = formatToolLabel(toolName);
        const elapsedMs = event.elapsed_ms;
        const completedContent = typeof elapsedMs === "number" ? `${toolLabel} (${elapsedMs.toFixed(1)}ms)` : toolLabel;
        upsertToolMessage(toolCallId, (existing) => ({
          id: existing?.id || `tool-${toolCallId}`,
          role: "tool",
          content: completedContent,
          toolCallId,
          toolStatus: "completed",
        }));
        return;
      }

      if (event.type === "final_answer") {
        const output = event.output;
        if (isRecord(output)) {
          const proposal = normalizeAiProposal(output.proposal);
          if (proposal) {
            setPendingProposal(proposal);
          } else {
            setPendingProposal(null);
          }
        }
        if (
          !streamedAnyAnswer &&
          runModel === "test" &&
          typeof output === "string" &&
          output.trim().toLowerCase() === TEST_MODEL_SENTINEL
        ) {
          appendAssistantContent(TEST_MODE_HINT);
          return;
        }

        if (!streamedAnyAnswer && typeof output === "string") {
          appendAssistantContent(output);
          return;
        }

        if (!streamedAnyAnswer && isRecord(output) && typeof output.answer === "string") {
          appendAssistantContent(output.answer);
        }
        return;
      }

      if (event.type === "run_failed" && typeof event.error === "string") {
        throw new Error(event.error);
      }
    };

    while (true) {
      const { done, value: streamValue } = await reader.read();
      if (done) break;

      buffer += decoder.decode(streamValue, { stream: true });
      const parts = buffer.split("\n\n");
      buffer = parts.pop() ?? "";

      for (const part of parts) {
        const parsedEvents = parseSseChunk(part);
        for (const event of parsedEvents) {
          await processEvent(event);
        }
      }
    }

    const trailingEvents = parseSseChunk(buffer);
    for (const trailingEvent of trailingEvents) {
      await processEvent(trailingEvent);
    }
    await waitForRenderLoopIdle();

    if (runModel === "test" && assistantContentSnapshot.trim().toLowerCase() === TEST_MODEL_SENTINEL) {
      replaceAssistantContent(TEST_MODE_HINT);
    }

    if (!streamedAnyAnswer) {
      setAiMessages((prev) =>
        prev.map((message) =>
          message.id === assistantMessageId
            ? {
                ...message,
                content:
                  runModel === "test" ? TEST_MODE_HINT : "I finished the run, but no text response was returned.",
              }
            : message,
        ),
      );
    }
  } catch (error) {
    setAiError(error instanceof Error ? error.message : GENERIC_FAILURE_MESSAGE);
    setAiMessages((prev) => {
      const existingIndex = prev.findIndex((message) => message.id === assistantMessageId);
      if (existingIndex < 0) {
        return pushAiMessages(prev, {
          id: `assistant-${Date.now()}`,
          role: "assistant",
          content: GENERIC_FAILURE_MESSAGE,
        });
      }

      const existingMessage = prev[existingIndex];
      if (existingMessage.role === "assistant" && existingMessage.content.trim().length === 0) {
        const updated = [...prev];
        updated[existingIndex] = {
          ...existingMessage,
          content: GENERIC_FAILURE_MESSAGE,
        };
        return trimAiMessages(updated);
      }

      return pushAiMessages(prev, {
        id: `assistant-${Date.now()}`,
        role: "assistant",
        content: GENERIC_FAILURE_MESSAGE,
      });
    });
  } finally {
    setIsGeneratingResponse(false);
  }
}
