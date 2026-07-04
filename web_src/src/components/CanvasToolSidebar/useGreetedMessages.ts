import { useContext, useMemo } from "react";
import { AccountContext } from "@/contexts/accountContextState";
import { getAgentBootInitialMessage } from "@/lib/agentBootContext";
import type { AgentMessage } from "./types";

// Prepend a synthetic greeting (and optional template intro) so it never disappears.
export function useGreetedMessages(rawMessages: AgentMessage[], canvasId: string): AgentMessage[] {
  const { account } = useContext(AccountContext);
  const greetingFirstName = account?.name?.split(" ")[0] ?? "there";
  // Read fresh each render so a /clear drops the intro once cleared for this canvas.
  const bootInitialMessage = getAgentBootInitialMessage(canvasId);

  return useMemo(() => {
    const greeting: AgentMessage = {
      id: "__greeting__",
      role: "assistant",
      content: `Hi ${greetingFirstName}! I'm your SuperPlane agent. I'll help you build and modify this canvas.`,
      createdAt: rawMessages[0]?.createdAt ?? null,
      toolCallId: "",
      toolName: "",
      toolStatus: "",
    };

    if (!bootInitialMessage) {
      return [greeting, ...rawMessages];
    }

    const templateIntro: AgentMessage = {
      id: "__boot_initial_message__",
      role: "assistant",
      content: bootInitialMessage,
      createdAt: rawMessages[0]?.createdAt ?? null,
      toolCallId: "",
      toolName: "",
      toolStatus: "",
    };

    return [greeting, templateIntro, ...rawMessages];
  }, [rawMessages, greetingFirstName, bootInitialMessage]);
}
