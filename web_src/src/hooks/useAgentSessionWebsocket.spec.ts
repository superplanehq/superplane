import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import { agentChatKeys, type AgentMessagesPage } from "@/hooks/useAgentChats";

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

function seedPages(queryClient: QueryClient, pages: AgentMessagesPage[]) {
  queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("session-1"), {
    pages,
    pageParams: pages.map(() => ""),
  });
}

function flatMessageIds(queryClient: QueryClient): string[] {
  const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("session-1"));
  if (!data) return [];
  return data.pages.flatMap((p) => p.messages.map((m) => m.id));
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

  it("appends an assistant_message to the newest page and fires onPersistedMessage", () => {
    const onPersisted = vi.fn();
    const queryClient = render({ onPersistedMessage: onPersisted });
    seedPages(queryClient, [
      {
        messages: [
          {
            id: "msg-existing",
            role: "user",
            content: "hi",
            toolName: "",
            toolCallId: "",
            toolStatus: "",
            createdAt: null,
          },
        ],
        hasMore: false,
      },
    ]);
    emit("assistant_message", {
      sessionId: "session-1",
      messageId: "msg-1",
      message: { id: "msg-1", role: "assistant", content: "done" },
    });
    expect(onPersisted).toHaveBeenCalledWith(expect.objectContaining({ id: "msg-1", content: "done" }));
    expect(flatMessageIds(queryClient)).toEqual(["msg-existing", "msg-1"]);
  });

  it("upserts tool events in place across pages", () => {
    const queryClient = render({});
    seedPages(queryClient, [{ messages: [], hasMore: false }]);
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
    const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("session-1"));
    expect(data?.pages[0].messages).toHaveLength(1);
    expect(data?.pages[0].messages[0].toolStatus).toBe("finished");
  });

  it("does not seed the cache when no pages have been fetched yet", () => {
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
