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
      createdAt: null,
      updatedAt: null,
    });
  });
});

describe("fromApiMessage", () => {
  it("returns null when id is missing", () => {
    expect(fromApiMessage({ role: "user", content: "hi" }, "chat-1", "org-1")).toBeNull();
  });

  it("preserves all populated fields", () => {
    const msg = fromApiMessage(
      {
        id: "msg-1",
        role: "assistant",
        content: "hello",
        toolName: "search",
        toolCallId: "call-1",
        toolStatus: "started",
        createdAt: "2026-05-13T00:00:00Z",
      },
      "chat-1",
      "org-1",
    );
    expect(msg).toEqual({
      id: "msg-1",
      role: "assistant",
      content: "hello",
      toolName: "search",
      toolCallId: "call-1",
      toolStatus: "started",
      images: [],
      createdAt: "2026-05-13T00:00:00Z",
    });
  });

  it("maps images to out-of-band URLs keyed by their original index", () => {
    const msg = fromApiMessage(
      {
        id: "msg-2",
        role: "user",
        content: "look",
        images: [{ mediaType: "MEDIA_TYPE_PNG" }, {}, { mediaType: "MEDIA_TYPE_JPEG" }],
      },
      "chat-9",
      "org-7",
    );
    expect(msg?.images).toEqual([
      { mediaType: "image/png", url: "/api/v1/agents/chats/chat-9/messages/msg-2/images/0?organization_id=org-7" },
      { mediaType: "image/jpeg", url: "/api/v1/agents/chats/chat-9/messages/msg-2/images/2?organization_id=org-7" },
    ]);
  });

  it("omits the organization query when no org is provided", () => {
    const msg = fromApiMessage(
      { id: "msg-3", role: "user", content: "look", images: [{ mediaType: "MEDIA_TYPE_PNG" }] },
      "chat-9",
      undefined,
    );
    expect(msg?.images?.[0].url).toBe("/api/v1/agents/chats/chat-9/messages/msg-3/images/0");
  });
});
