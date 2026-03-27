import type { Dispatch, SetStateAction } from "react";
import type { AiBuilderMessage, AiBuilderProposal } from "./agentChat";
import { normalizeAiProposal } from "./agentChatProposal";

type JsonObject = Record<string, unknown>;

export type ChatStreamEvent =
  | { type: "run_started"; model?: string }
  | { type: "model_delta"; content?: string }
  | { type: "tool_started"; tool_name?: string; tool_call_id?: string }
  | { type: "tool_finished"; tool_name?: string; tool_call_id?: string; elapsed_ms?: number }
  | { type: "final_answer"; output?: unknown }
  | { type: "run_failed"; error?: string }
  | { type: "run_completed" }
  | { type: "done" }
  | { type: "raw_data"; content: string };

type InsertAiMessageBefore = (
  previous: AiBuilderMessage[],
  next: AiBuilderMessage,
  beforeId: string,
) => AiBuilderMessage[];

type TrimAiMessages = (messages: AiBuilderMessage[]) => AiBuilderMessage[];

type StreamOutcome = {
  assistantContentSnapshot: string;
  streamedAnyAnswer: boolean;
  runModel: string;
};

type StreamState = {
  runModel: string;
  streamedAnyAnswer: boolean;
};

type StreamController = {
  appendAssistantContent: (chunk: string) => void;
  upsertToolMessage: (event: Extract<ChatStreamEvent, { type: "tool_started" | "tool_finished" }>) => void;
  waitForRenderLoopIdle: () => Promise<void>;
  getAssistantContentSnapshot: () => string;
};

function isRecord(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeRunStartedEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "run_started",
    model: typeof value.model === "string" ? value.model : undefined,
  };
}

function normalizeModelDeltaEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "model_delta",
    content: typeof value.content === "string" ? value.content : undefined,
  };
}

function normalizeToolStartedEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "tool_started",
    tool_name: typeof value.tool_name === "string" ? value.tool_name : undefined,
    tool_call_id: typeof value.tool_call_id === "string" ? value.tool_call_id : undefined,
  };
}

function normalizeToolFinishedEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "tool_finished",
    tool_name: typeof value.tool_name === "string" ? value.tool_name : undefined,
    tool_call_id: typeof value.tool_call_id === "string" ? value.tool_call_id : undefined,
    elapsed_ms: typeof value.elapsed_ms === "number" ? value.elapsed_ms : undefined,
  };
}

function normalizeFinalAnswerEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "final_answer",
    output: value.output,
  };
}

function normalizeRunFailedEvent(value: JsonObject): ChatStreamEvent {
  return {
    type: "run_failed",
    error: typeof value.error === "string" ? value.error : undefined,
  };
}

function normalizeStreamEvent(value: unknown): ChatStreamEvent | null {
  if (!isRecord(value) || typeof value.type !== "string") {
    return null;
  }

  switch (value.type) {
    case "run_started":
      return normalizeRunStartedEvent(value);
    case "model_delta":
      return normalizeModelDeltaEvent(value);
    case "tool_started":
      return normalizeToolStartedEvent(value);
    case "tool_finished":
      return normalizeToolFinishedEvent(value);
    case "final_answer":
      return normalizeFinalAnswerEvent(value);
    case "run_failed":
      return normalizeRunFailedEvent(value);
    case "run_completed":
      return { type: "run_completed" };
    case "done":
      return { type: "done" };
    default:
      return null;
  }
}

function extractDataLines(chunk: string): string[] {
  return chunk
    .split("\n")
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.replace(/^data:\s*/, ""));
}

export function parseSseChunk(rawChunk: string): ChatStreamEvent[] {
  const events: ChatStreamEvent[] = [];

  for (const chunk of rawChunk.split("\n\n")) {
    const dataLines = extractDataLines(chunk);
    if (!dataLines.length) {
      continue;
    }

    const merged = dataLines.join("\n").trim();
    if (!merged) {
      continue;
    }

    try {
      const normalized = normalizeStreamEvent(JSON.parse(merged));
      if (normalized) {
        events.push(normalized);
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

function createToolCallId(toolName: string, toolCallId?: string): string {
  return typeof toolCallId === "string" && toolCallId.trim().length > 0 ? toolCallId : `${toolName}-${Date.now()}`;
}

function createAssistantStreamController({
  assistantMessageId,
  formatToolLabel,
  insertAiMessageBefore,
  setAiMessages,
  trimAiMessages,
}: {
  assistantMessageId: string;
  formatToolLabel: (toolName: string) => string;
  insertAiMessageBefore: InsertAiMessageBefore;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  trimAiMessages: TrimAiMessages;
}): StreamController {
  let assistantContentSnapshot = "";
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
        setAiMessages((previous) =>
          previous.map((message) =>
            message.id === assistantMessageId ? { ...message, content: `${message.content}${nextChunk}` } : message,
          ),
        );
        await sleep(8);
      }
    } finally {
      isRenderLoopRunning = false;
    }
  };

  const appendAssistantContent = (chunk: string) => {
    if (!chunk) {
      return;
    }

    pendingRenderBuffer += chunk;
    void flushPendingRenderBuffer();
  };

  const upsertToolMessage = (event: Extract<ChatStreamEvent, { type: "tool_started" | "tool_finished" }>) => {
    const toolName = typeof event.tool_name === "string" ? event.tool_name : "unknown";
    const toolCallId = createToolCallId(toolName, event.tool_call_id);
    const toolLabel = formatToolLabel(toolName);
    const content =
      event.type === "tool_started"
        ? `${toolLabel}...`
        : typeof event.elapsed_ms === "number"
          ? `${toolLabel} (${event.elapsed_ms.toFixed(1)}ms)`
          : toolLabel;
    const toolStatus = event.type === "tool_started" ? "running" : "completed";

    setAiMessages((previous) => {
      const existingIndex = previous.findIndex(
        (message) => message.role === "tool" && message.toolCallId === toolCallId,
      );
      const nextMessage: AiBuilderMessage = {
        id: existingIndex >= 0 ? previous[existingIndex].id : `tool-${toolCallId}`,
        role: "tool",
        content,
        toolCallId,
        toolStatus,
      };
      if (existingIndex >= 0) {
        const updated = [...previous];
        updated[existingIndex] = nextMessage;
        return trimAiMessages(updated);
      }

      return insertAiMessageBefore(previous, nextMessage, assistantMessageId);
    });
  };

  const waitForRenderLoopIdle = async () => {
    while (isRenderLoopRunning || pendingRenderBuffer.length > 0) {
      await sleep(10);
    }
  };

  return {
    appendAssistantContent,
    upsertToolMessage,
    waitForRenderLoopIdle,
    getAssistantContentSnapshot: () => assistantContentSnapshot,
  };
}

function appendFinalAnswerContent({
  output,
  state,
  appendAssistantContent,
  setPendingProposal,
  testModeHint,
  testModelSentinel,
}: {
  output: unknown;
  state: StreamState;
  appendAssistantContent: (chunk: string) => void;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  testModeHint: string;
  testModelSentinel: string;
}): void {
  if (isRecord(output)) {
    setPendingProposal(normalizeAiProposal(output.proposal));
  }

  if (state.streamedAnyAnswer) {
    return;
  }

  if (
    state.runModel === "test" &&
    typeof output === "string" &&
    output.trim().toLowerCase() === testModelSentinel.toLowerCase()
  ) {
    state.streamedAnyAnswer = true;
    appendAssistantContent(testModeHint);
    return;
  }

  if (typeof output === "string") {
    state.streamedAnyAnswer = true;
    appendAssistantContent(output);
    return;
  }

  if (isRecord(output) && typeof output.answer === "string") {
    state.streamedAnyAnswer = true;
    appendAssistantContent(output.answer);
  }
}

async function processChatStreamEvent({
  event,
  state,
  controller,
  setPendingProposal,
  testModeHint,
  testModelSentinel,
}: {
  event: ChatStreamEvent;
  state: StreamState;
  controller: StreamController;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  testModeHint: string;
  testModelSentinel: string;
}): Promise<void> {
  switch (event.type) {
    case "run_started":
      state.runModel = typeof event.model === "string" ? event.model.trim().toLowerCase() : "";
      return;
    case "model_delta":
      if (typeof event.content === "string" && event.content.length > 0) {
        state.streamedAnyAnswer = true;
        controller.appendAssistantContent(event.content);
      }
      return;
    case "tool_started":
    case "tool_finished":
      controller.upsertToolMessage(event);
      return;
    case "final_answer":
      appendFinalAnswerContent({
        output: event.output,
        state,
        appendAssistantContent: controller.appendAssistantContent,
        setPendingProposal,
        testModeHint,
        testModelSentinel,
      });
      return;
    case "run_failed":
      if (typeof event.error === "string") {
        throw new Error(event.error);
      }
      return;
    default:
      return;
  }
}

async function readResponseEvents(
  response: Response,
  onEvent: (event: ChatStreamEvent) => Promise<void>,
): Promise<void> {
  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error("Response body is not available.");
  }

  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n\n");
    buffer = parts.pop() ?? "";

    for (const part of parts) {
      for (const event of parseSseChunk(part)) {
        await onEvent(event);
      }
    }
  }

  for (const event of parseSseChunk(buffer)) {
    await onEvent(event);
  }
}

export async function consumeChatResponseStream({
  assistantMessageId,
  formatToolLabel,
  insertAiMessageBefore,
  response,
  setAiMessages,
  setPendingProposal,
  testModeHint,
  testModelSentinel,
  trimAiMessages,
}: {
  assistantMessageId: string;
  formatToolLabel: (toolName: string) => string;
  insertAiMessageBefore: InsertAiMessageBefore;
  response: Response;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  testModeHint: string;
  testModelSentinel: string;
  trimAiMessages: TrimAiMessages;
}): Promise<StreamOutcome> {
  const controller = createAssistantStreamController({
    assistantMessageId,
    formatToolLabel,
    insertAiMessageBefore,
    setAiMessages,
    trimAiMessages,
  });
  const state: StreamState = {
    runModel: "",
    streamedAnyAnswer: false,
  };

  await readResponseEvents(response, async (event) => {
    await processChatStreamEvent({
      event,
      state,
      controller,
      setPendingProposal,
      testModeHint,
      testModelSentinel,
    });
  });
  await controller.waitForRenderLoopIdle();

  return {
    assistantContentSnapshot: controller.getAssistantContentSnapshot(),
    streamedAnyAnswer: state.streamedAnyAnswer,
    runModel: state.runModel,
  };
}
