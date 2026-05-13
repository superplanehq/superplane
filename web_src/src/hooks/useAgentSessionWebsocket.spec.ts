import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import { agentChatKeys } from "@/hooks/useAgentChats";

const { useWebSocketMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
}));

vi.mock("react-use-websocket", () => ({
  default: useWebSocketMock,
}));

import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";

afterEach(() => {
  vi.clearAllMocks();
});

function lastCall() {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call) throw new Error("useWebSocket was not invoked");
  return call;
}

function emit(event: string, payload: unknown) {
  const [, options] = lastCall();
  const onMessage = options.onMessage as (e: MessageEvent<unknown>) => void;
  act(() => {
    onMessage(
      new MessageEvent("message", {
        data: JSON.stringify({ ...(payload as object), event }),
      }),
    );
  });
}

function render(callbacks: Parameters<typeof useAgentSessionWebsocket>[2]) {
  const queryClient = new QueryClient();
  renderHook(() => useAgentSessionWebsocket("session-1", "org-1", callbacks), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children),
  });
  return queryClient;
}

describe("useAgentSessionWebsocket", () => {
  it("connects to the per-session WebSocket URL with org id", () => {
    render({});
    const [url, , enabled] = lastCall();
    expect(url).toContain("/ws/agents/sessions/session-1");
    expect(url).toContain("organization_id=org-1");
    expect(enabled).toBe(true);
  });

  it("disables the connection when no session id is provided", () => {
    const queryClient = new QueryClient();
    renderHook(() => useAgentSessionWebsocket(null, "org-1", {}), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });
    const [url, , enabled] = lastCall();
    expect(url).toBeNull();
    expect(enabled).toBe(false);
  });

  it("forwards assistant_delta payloads via onAssistantDelta", () => {
    const onDelta = vi.fn();
    render({ onAssistantDelta: onDelta });
    emit("assistant_delta", { sessionId: "session-1", extra: { text: "Hello" } });
    expect(onDelta).toHaveBeenCalledWith("Hello");
  });

  it("appends an assistant_message into the cached list and fires onPersistedMessage", () => {
    const onPersisted = vi.fn();
    const queryClient = render({ onPersistedMessage: onPersisted });
    // Seed the cache so the upsert can apply; the hook deliberately
    // doesn't seed from WS alone (the initial list comes from the API).
    queryClient.setQueryData(agentChatKeys.messages("session-1"), [
      {
        id: "msg-existing",
        role: "user",
        content: "hi",
        toolName: "",
        toolCallId: "",
        toolStatus: "",
        createdAt: null,
      },
    ]);
    emit("assistant_message", {
      sessionId: "session-1",
      messageId: "msg-1",
      message: { id: "msg-1", role: "assistant", content: "done" },
    });
    expect(onPersisted).toHaveBeenCalledWith(expect.objectContaining({ id: "msg-1", content: "done" }));
    const cached = queryClient.getQueryData(agentChatKeys.messages("session-1")) as Array<{ id: string }>;
    expect(cached.map((m) => m.id)).toEqual(["msg-existing", "msg-1"]);
  });

  it("upserts tool events into the cache (started -> finished updates in place)", () => {
    const queryClient = render({});
    queryClient.setQueryData(agentChatKeys.messages("session-1"), []);
    emit("tool_started", {
      sessionId: "session-1",
      messageId: "tool-1",
      message: { id: "tool-1", role: "tool", toolName: "search", toolStatus: "started" },
    });
    emit("tool_finished", {
      sessionId: "session-1",
      messageId: "tool-1",
      message: { id: "tool-1", role: "tool", toolName: "search", toolStatus: "finished" },
    });
    const cached = queryClient.getQueryData(agentChatKeys.messages("session-1")) as Array<{
      id: string;
      toolStatus: string;
    }>;
    expect(cached).toHaveLength(1);
    expect(cached[0].toolStatus).toBe("finished");
  });

  it("does not seed the cache when the messages query has not been fetched yet", () => {
    const queryClient = render({});
    emit("assistant_message", {
      sessionId: "session-1",
      messageId: "msg-1",
      message: { id: "msg-1", role: "assistant", content: "hi" },
    });
    expect(queryClient.getQueryData(agentChatKeys.messages("session-1"))).toBeUndefined();
  });

  it("forwards status changes to onStatusChange", () => {
    const onStatus = vi.fn();
    render({ onStatusChange: onStatus });
    emit("stream_started", { sessionId: "session-1", status: "streaming" });
    emit("turn_completed", { sessionId: "session-1", status: "idle" });
    emit("session_failed", { sessionId: "session-1", status: "failed", error: "boom" });
    expect(onStatus).toHaveBeenNthCalledWith(1, "streaming", undefined);
    expect(onStatus).toHaveBeenNthCalledWith(2, "idle", undefined);
    expect(onStatus).toHaveBeenNthCalledWith(3, "failed", "boom");
  });

  it("ignores malformed payloads without crashing", () => {
    const onDelta = vi.fn();
    render({ onAssistantDelta: onDelta });
    const [, options] = lastCall();
    const onMessage = options.onMessage as (e: MessageEvent<unknown>) => void;
    act(() => {
      onMessage(new MessageEvent("message", { data: "not-json" }));
    });
    expect(onDelta).not.toHaveBeenCalled();
  });
});
