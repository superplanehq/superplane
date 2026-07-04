import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useThinkingIndicator } from "./agentConversationState";
import type { AgentMessage } from "./types";

function message(overrides: Partial<AgentMessage>): AgentMessage {
  return {
    id: "message-1",
    role: "assistant",
    content: "",
    toolName: "",
    toolCallId: "",
    toolStatus: "",
    createdAt: null,
    ...overrides,
  };
}

function thinking(messages: AgentMessage[], status = "streaming") {
  return renderHook(() => useThinkingIndicator(messages, status)).result.current;
}

describe("useThinkingIndicator", () => {
  it("shows while the agent is streaming without an active tool", () => {
    expect(thinking([message({ id: "user-1", role: "user", content: "Run this" })])).toBe(true);
  });

  it("keeps showing after a tool finishes while the agent is still streaming", () => {
    expect(
      thinking([
        message({ id: "user-1", role: "user", content: "Run this" }),
        message({ id: "tool-start-1", role: "tool", toolCallId: "call-1", toolStatus: "started" }),
        message({ id: "tool-1", role: "tool", toolCallId: "call-1", toolStatus: "finished" }),
      ]),
    ).toBe(true);
  });

  it("hides while a tool is active", () => {
    expect(
      thinking([
        message({ id: "user-1", role: "user", content: "Run this" }),
        message({ id: "tool-1", role: "tool", toolCallId: "call-1", toolStatus: "started" }),
      ]),
    ).toBe(false);
  });

  it("hides when the session is not streaming", () => {
    expect(thinking([message({ id: "user-1", role: "user", content: "Run this" })], "idle")).toBe(false);
  });
});
