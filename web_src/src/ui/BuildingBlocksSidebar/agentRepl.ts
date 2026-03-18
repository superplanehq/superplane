import type { Dispatch, SetStateAction } from "react";
import type { AiCanvasOperation } from "./index";

export type AiBuilderMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
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

    const appendAssistantContent = (chunk: string) => {
      if (!chunk) return;
      assistantContentSnapshot += chunk;
      streamedAnyAnswer = true;
      setAiMessages((prev) =>
        prev.map((message) =>
          message.id === assistantMessageId ? { ...message, content: `${message.content}${chunk}` } : message,
        ),
      );
    };

    const replaceAssistantContent = (content: string) => {
      assistantContentSnapshot = content;
      streamedAnyAnswer = true;
      setAiMessages((prev) =>
        prev.map((message) => (message.id === assistantMessageId ? { ...message, content } : message)),
      );
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
          if (event.type === "run_started" && typeof event.model === "string") {
            runModel = event.model.trim().toLowerCase();
            continue;
          }

          if (event.type === "model_delta" && typeof event.content === "string") {
            appendAssistantContent(event.content);
            continue;
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
              continue;
            }
            if (!streamedAnyAnswer && typeof output === "string") {
              appendAssistantContent(output);
              continue;
            }

            if (
              !streamedAnyAnswer &&
              output &&
              typeof output === "object" &&
              typeof (output as { answer?: unknown }).answer === "string"
            ) {
              appendAssistantContent((output as { answer: string }).answer);
            }
            continue;
          }

          if (event.type === "run_failed" && typeof event.error === "string") {
            throw new Error(event.error);
          }
        }
      }
    }

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
