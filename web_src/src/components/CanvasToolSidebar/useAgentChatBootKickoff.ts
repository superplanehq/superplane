import { useEffect, useRef, useState } from "react";
import { createSystemMessage } from "@/components/AgentSidebar/systemMessages";
import type { useAgentChatMessages, useSendAgentChatMessage } from "@/hooks/useAgentChats";
import {
  AGENT_BOOT_CONTEXT_READY_EVENT,
  clearAgentBootContext,
  getAgentBootMessage,
  isAgentBootReady,
} from "@/lib/agentBootContext";

export function useAgentChatBootKickoff({
  messagesQuery,
  sendMutation,
  chatId,
  canvasId,
  isAutoLayoutOnUpdateEnabled,
}: {
  messagesQuery: ReturnType<typeof useAgentChatMessages>;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  chatId: string;
  canvasId: string;
  isAutoLayoutOnUpdateEnabled: boolean;
}) {
  const [bootReadinessSignal, setBootReadinessSignal] = useState(0);
  const bootState = useRef<"idle" | "sending" | "sent">("idle");

  useEffect(() => {
    const handleBootReady = (event: Event) => {
      const detail = (event as CustomEvent<{ canvasId?: string }>).detail;
      if (detail?.canvasId === canvasId) {
        setBootReadinessSignal((current) => current + 1);
      }
    };

    window.addEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, handleBootReady);
    return () => window.removeEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, handleBootReady);
  }, [canvasId]);

  useEffect(() => {
    if (bootState.current !== "idle") return;
    if (!messagesQuery.data || messagesQuery.isLoading) return;
    if (!isAgentBootReady(canvasId)) return;

    const allMessages = messagesQuery.data.pages?.flatMap((p) => p.messages) ?? [];
    if (allMessages.length > 0) return;

    const bootMessage = getAgentBootMessage(canvasId);
    if (!bootMessage) {
      bootState.current = "sent";
      return;
    }

    bootState.current = "sending";
    void sendMutation
      .mutateAsync({
        chatId,
        content: createSystemMessage(bootMessage),
        autoLayoutOnUpdateEnabled: isAutoLayoutOnUpdateEnabled,
      })
      .then(() => {
        bootState.current = "sent";
        clearAgentBootContext();
      })
      .catch(() => {
        bootState.current = "idle";
      });
  }, [
    messagesQuery.data,
    messagesQuery.isLoading,
    bootReadinessSignal,
    chatId,
    canvasId,
    isAutoLayoutOnUpdateEnabled,
    sendMutation,
  ]);
}
