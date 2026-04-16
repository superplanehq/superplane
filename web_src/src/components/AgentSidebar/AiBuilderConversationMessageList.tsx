import { AiMessage } from "./AiBuilderChatMessage";
import { FinishedToolCallsCollapsible } from "./FinishedToolCallsCollapsible";
import type { AiBuilderMessage } from "./agentChat";
import { useMemo, type ReactNode } from "react";

export type AiBuilderConversationMessageListProps = {
  messages: AiBuilderMessage[];
  isGeneratingResponse: boolean;
};

export function AiBuilderConversationMessageList({
  messages,
  isGeneratingResponse,
}: AiBuilderConversationMessageListProps) {
  const activeToolTurnBounds = useMemo(() => {
    let lastUserIndex = -1;
    let lastAssistantIndex = -1;
    for (let i = 0; i < messages.length; i++) {
      if (messages[i].role === "user") {
        lastUserIndex = i;
      }
      if (messages[i].role === "assistant") {
        lastAssistantIndex = i;
      }
    }
    return { lastUserIndex, lastAssistantIndex };
  }, [messages]);

  const conversationItems = useMemo(
    () =>
      buildAiConversationItems({
        messages,
        isGeneratingResponse,
        lastUserIndex: activeToolTurnBounds.lastUserIndex,
        lastAssistantIndex: activeToolTurnBounds.lastAssistantIndex,
      }),
    [messages, isGeneratingResponse, activeToolTurnBounds.lastUserIndex, activeToolTurnBounds.lastAssistantIndex],
  );

  return <>{conversationItems}</>;
}

function shouldShowToolCallForActiveTurn({
  messageIndex,
  isGeneratingResponse,
  lastUserIndex,
  lastAssistantIndex,
}: {
  messageIndex: number;
  isGeneratingResponse: boolean;
  lastUserIndex: number;
  lastAssistantIndex: number;
}): boolean {
  if (!isGeneratingResponse) {
    return false;
  }
  if (lastUserIndex < 0 || lastAssistantIndex <= lastUserIndex) {
    return false;
  }
  return messageIndex > lastUserIndex && messageIndex < lastAssistantIndex;
}

type BuildAiConversationItemsParams = {
  messages: AiBuilderMessage[];
  isGeneratingResponse: boolean;
  lastUserIndex: number;
  lastAssistantIndex: number;
};

function buildAiConversationItems({
  messages,
  isGeneratingResponse,
  lastUserIndex,
  lastAssistantIndex,
}: BuildAiConversationItemsParams): ReactNode[] {
  const items: ReactNode[] = [];
  let i = 0;
  while (i < messages.length) {
    const message = messages[i];
    if (message.role === "tool") {
      const groupStart = i;
      let j = i;
      while (j < messages.length && messages[j].role === "tool") {
        j += 1;
      }
      const showToolsInlineForLiveTurn =
        isGeneratingResponse &&
        Array.from({ length: j - groupStart }, (_, offset) => groupStart + offset).every((messageIndex) =>
          shouldShowToolCallForActiveTurn({
            messageIndex,
            isGeneratingResponse,
            lastUserIndex,
            lastAssistantIndex,
          }),
        );

      if (showToolsInlineForLiveTurn) {
        for (let k = groupStart; k < j; k++) {
          const toolMessage = messages[k];
          items.push(<AiMessage key={toolMessage.id} message={toolMessage} animate />);
        }
      } else {
        const toolGroup = messages.slice(groupStart, j);
        const groupKey = toolGroup.map((m) => m.id).join(":") || `tools-${groupStart}`;
        items.push(<FinishedToolCallsCollapsible key={groupKey} tools={toolGroup} />);
      }
      i = j;
      continue;
    }
    items.push(<AiMessage key={message.id} message={message} />);
    i += 1;
  }
  return items;
}
