import { describe, expect, it } from "vitest";
import { fromApiChat, fromApiMessage } from "./types";

describe("fromApiChat", () => {
  it("returns null when input is undefined", () => {
    expect(fromApiChat(undefined)).toBeNull();
  });

  it("returns null when id is missing — the row is unusable without it", () => {
    expect(fromApiChat({ canvasId: "c-1" })).toBeNull();
  });

  it("narrows optional fields to non-null defaults", () => {
    const chat = fromApiChat({ id: "chat-1" });
    expect(chat).toEqual({
      id: "chat-1",
      canvasId: "",
      provider: "",
      status: "idle",
      title: "",
      createdAt: null,
      updatedAt: null,
      archivedAt: null,
    });
  });
});

describe("fromApiMessage", () => {
  it("returns null when id is missing", () => {
    expect(fromApiMessage({ role: "user", content: "hi" })).toBeNull();
  });

  it("preserves all populated fields", () => {
    const msg = fromApiMessage({
      id: "msg-1",
      role: "assistant",
      content: "hello",
      toolName: "search",
      toolCallId: "call-1",
      toolStatus: "started",
      createdAt: "2026-05-13T00:00:00Z",
    });
    expect(msg).toEqual({
      id: "msg-1",
      role: "assistant",
      content: "hello",
      toolName: "search",
      toolCallId: "call-1",
      toolStatus: "started",
      createdAt: "2026-05-13T00:00:00Z",
    });
  });
});
