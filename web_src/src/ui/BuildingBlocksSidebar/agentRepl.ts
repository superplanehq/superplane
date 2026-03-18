import type { Dispatch, SetStateAction } from "react";
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

type ReplStreamEvent = {
  type?: string;
  [key: string]: unknown;
};

const AI_HISTORY_RECENT_TURNS = 8;
const AI_HISTORY_OLDER_TURNS = 6;
const AI_HISTORY_MAX_MESSAGE_CHARS = 320;
const AI_MAX_STORED_MESSAGES = 50;
const TEST_MODEL_SENTINEL = "success (no tool calls)";
const TEST_MODE_HINT =
  "Agent is running in test mode. Set AI_MODEL in agent/.env to a real model and configure agent credentials to get canvas-aware answers.";
const GENERIC_FAILURE_MESSAGE = "I couldn't generate changes right now. Please try again.";

function compactMessageContent(content: string): string {
  const normalized = content.replace(/\s+/g, " ").trim();
  if (normalized.length <= AI_HISTORY_MAX_MESSAGE_CHARS) {
    return normalized;
  }

  return `${normalized.slice(0, AI_HISTORY_MAX_MESSAGE_CHARS)}...`;
}

function formatConversationTurns(messages: AiBuilderMessage[]): string[] {
  return messages
    .filter((message) => message.role === "user" || message.role === "assistant")
    .map((message) => `${message.role}: ${compactMessageContent(message.content)}`)
    .filter((line) => line.length > 0);
}

function buildPromptWithConversationContext(messages: AiBuilderMessage[], prompt: string): string {
  const turns = formatConversationTurns(messages);
  if (turns.length === 0) {
    return prompt;
  }

  const recentTurns = turns.slice(-AI_HISTORY_RECENT_TURNS);
  const olderTurns = turns.slice(0, -AI_HISTORY_RECENT_TURNS).slice(-AI_HISTORY_OLDER_TURNS);
  const contextSections = [
    "Conversation context (use this for continuity and intent resolution):",
    ...(olderTurns.length > 0 ? [`Earlier turns summary:\n${olderTurns.join("\n")}`] : []),
    `Recent turns:\n${recentTurns.join("\n")}`,
    `Current user request:\n${prompt}`,
  ];

  return contextSections.join("\n\n");
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

function parseSseChunk(rawChunk: string): ReplStreamEvent[] {
  const chunks = rawChunk.split("\n\n");
  const events: ReplStreamEvent[] = [];

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
      if (parsed && typeof parsed === "object") {
        events.push(parsed as ReplStreamEvent);
      }
    } catch {
      events.push({ type: "raw_data", content: merged });
    }
  }

  return events;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

type SendAgentReplPromptArgs = {
  value?: string;
  aiInput: string;
  aiMessages: AiBuilderMessage[];
  canvasId?: string;
  organizationId?: string;
  agentReplWebUrl: string;
  isGeneratingResponse: boolean;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  focusInput: () => void;
};

export async function sendAgentReplPrompt({
  value,
  aiInput,
  aiMessages,
  canvasId,
  organizationId,
  agentReplWebUrl,
  isGeneratingResponse,
  setAiMessages,
  setAiInput,
  setAiError,
  setIsGeneratingResponse,
  setPendingProposal,
  focusInput,
}: SendAgentReplPromptArgs): Promise<void> {
  const nextPrompt = (value ?? aiInput).trim();
  if (!nextPrompt || isGeneratingResponse || !canvasId) {
    return;
  }

  if (nextPrompt.toLowerCase() === "/clear") {
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
    requestAnimationFrame(() => {
      focusInput();
    });
    return;
  }

  const contextualPrompt = buildPromptWithConversationContext(aiMessages, nextPrompt);

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

  try {
    const assistantMessageId = `assistant-${Date.now()}`;
    setAiMessages((prev) =>
      pushAiMessages(prev, {
        id: assistantMessageId,
        role: "assistant",
        content: "",
      }),
    );
    setPendingProposal(null);

    const response = await fetch(`${agentReplWebUrl}/v1/repl/stream`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
      },
      body: JSON.stringify({
        question: contextualPrompt,
        canvas_id: canvasId,
        org_id: organizationId || undefined,
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

    const processEvent = async (event: ReplStreamEvent) => {
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

        if (
          !streamedAnyAnswer &&
          output &&
          typeof output === "object" &&
          typeof (output as { answer?: unknown }).answer === "string"
        ) {
          appendAssistantContent((output as { answer: string }).answer);
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
    setAiMessages((prev) =>
      pushAiMessages(prev, {
        id: `assistant-${Date.now()}`,
        role: "assistant",
        content: GENERIC_FAILURE_MESSAGE,
      }),
    );
  } finally {
    setIsGeneratingResponse(false);
  }
}
