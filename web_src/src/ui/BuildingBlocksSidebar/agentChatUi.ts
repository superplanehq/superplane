import type { Dispatch, SetStateAction } from "react";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";

export const GENERIC_FAILURE_MESSAGE = "I couldn't generate changes right now. Please try again.";

/**
 * Ensures only clean, human-readable messages reach the UI.
 * Rejects strings that look like raw exception reprs (contain braces, tracebacks,
 * or exceed a reasonable display length) since those are never actionable for users.
 */
export function sanitizeErrorMessage(error: unknown): string {
  if (!(error instanceof Error)) return GENERIC_FAILURE_MESSAGE;

  const message = error.message;
  if (!message || message.length > 200 || message.includes("{") || message.includes("Traceback")) {
    return GENERIC_FAILURE_MESSAGE;
  }

  return message;
}

type SendChatPromptUiArgs = {
  focusInput: () => void;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
};

export function clearChatPrompt({
  focusInput,
  setAiError,
  setAiInput,
  setAiMessages,
  setCurrentChatId,
  setPendingProposal,
}: SendChatPromptUiArgs): void {
  setAiMessages([]);
  setCurrentChatId(null);
  setPendingProposal(null);
  setAiError(null);
  setAiInput("");
  requestAnimationFrame(() => {
    focusInput();
  });
}

export function addLocalPromptMessages({
  assistantMessageId,
  focusInput,
  nextPrompt,
  pushAiMessages,
  setAiError,
  setAiInput,
  setAiMessages,
  setIsGeneratingResponse,
}: {
  assistantMessageId: string;
  focusInput: () => void;
  nextPrompt: string;
  pushAiMessages: (previous: AiBuilderMessage[], next: AiBuilderMessage | AiBuilderMessage[]) => AiBuilderMessage[];
  setAiError: Dispatch<SetStateAction<string | null>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
}): void {
  setAiMessages((previous) =>
    pushAiMessages(previous, {
      id: `user-${Date.now()}`,
      role: "user",
      content: nextPrompt,
    }),
  );
  setAiInput("");
  requestAnimationFrame(() => {
    focusInput();
  });
  setAiError(null);
  setIsGeneratingResponse(true);
  setAiMessages((previous) =>
    pushAiMessages(previous, {
      id: assistantMessageId,
      role: "assistant",
      content: "",
    }),
  );
}

export function applyChatPromptFailure({
  assistantMessageId,
  error,
  pushAiMessages,
  setAiError,
  setAiMessages,
  trimAiMessages,
}: {
  assistantMessageId: string;
  error: unknown;
  pushAiMessages: (previous: AiBuilderMessage[], next: AiBuilderMessage | AiBuilderMessage[]) => AiBuilderMessage[];
  setAiError: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  trimAiMessages: (messages: AiBuilderMessage[]) => AiBuilderMessage[];
}): void {
  setAiError(sanitizeErrorMessage(error));
  setAiMessages((previous) => {
    const existingIndex = previous.findIndex((message) => message.id === assistantMessageId);
    if (existingIndex < 0) {
      return pushAiMessages(previous, {
        id: `assistant-${Date.now()}`,
        role: "assistant",
        content: GENERIC_FAILURE_MESSAGE,
      });
    }

    const existingMessage = previous[existingIndex];
    if (existingMessage.role === "assistant" && existingMessage.content.trim().length === 0) {
      const updated = [...previous];
      updated[existingIndex] = {
        ...existingMessage,
        content: GENERIC_FAILURE_MESSAGE,
      };
      return trimAiMessages(updated);
    }

    return pushAiMessages(previous, {
      id: `assistant-${Date.now()}`,
      role: "assistant",
      content: GENERIC_FAILURE_MESSAGE,
    });
  });
}

export function applyStreamOutcome({
  assistantContentSnapshot,
  assistantMessageId,
  runModel,
  setAiMessages,
  streamedAnyAnswer,
  testModeHint,
  testModelSentinel,
}: {
  assistantContentSnapshot: string;
  assistantMessageId: string;
  runModel: string;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  streamedAnyAnswer: boolean;
  testModeHint: string;
  testModelSentinel: string;
}): void {
  if (runModel === "test" && assistantContentSnapshot.trim().toLowerCase() === testModelSentinel.toLowerCase()) {
    setAiMessages((previous) =>
      previous.map((message) => (message.id === assistantMessageId ? { ...message, content: testModeHint } : message)),
    );
  }

  if (streamedAnyAnswer) {
    return;
  }

  setAiMessages((previous) =>
    previous.map((message) =>
      message.id === assistantMessageId
        ? {
            ...message,
            content: runModel === "test" ? testModeHint : "I finished the run, but no text response was returned.",
          }
        : message,
    ),
  );
}

export function prependChatSession({
  chatId,
  nextPrompt,
  setChatSessions,
}: {
  chatId: string;
  nextPrompt: string;
  setChatSessions?: Dispatch<SetStateAction<AiChatSession[]>>;
}): void {
  if (!setChatSessions) {
    return;
  }

  const createdAt = new Date().toISOString();
  setChatSessions((previousSessions) => [
    {
      id: chatId,
      title: nextPrompt,
      initialMessage: nextPrompt,
      createdAt,
    },
    ...previousSessions.filter((session) => session.id !== chatId),
  ]);
}
