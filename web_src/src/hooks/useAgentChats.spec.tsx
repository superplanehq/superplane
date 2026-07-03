import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { agentsListAgentChatMessages, agentsSendAgentChatMessage } from "@/api-client/sdk.gen";
import type { AgentMessagesPage } from "./useAgentChats";
import { agentChatKeys, useAgentChatMessages, useResetCanvasAgentChat, useSendAgentChatMessage } from "./useAgentChats";

vi.mock("@/api-client/sdk.gen", () => ({
  agentsListAgentChatMessages: vi.fn(),
  agentsSendAgentChatMessage: vi.fn(),
}));

function wrapper(queryClient: QueryClient) {
  return function TestQueryClientProvider({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useSendAgentChatMessage", () => {
  it("adds a local user message before the send request resolves", async () => {
    type SendAgentChatMessageResult = Awaited<ReturnType<typeof agentsSendAgentChatMessage>>;
    let resolveSend: ((value: SendAgentChatMessageResult) => void) | undefined;
    vi.mocked(agentsSendAgentChatMessage).mockReturnValue(
      new Promise((resolve) => {
        resolveSend = resolve;
      }) as ReturnType<typeof agentsSendAgentChatMessage>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [{ messages: [], hasMore: false }],
      pageParams: [""],
    });

    const { result } = renderHook(() => useSendAgentChatMessage("org-1", "canvas-1"), {
      wrapper: wrapper(queryClient),
    });

    const sendPromise = result.current.mutateAsync({ chatId: "chat-1", content: "Build this", mode: "builder" });

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages[0]?.content).toBe("Build this");
    });

    if (!resolveSend) {
      throw new Error("send promise was not created");
    }

    resolveSend({
      data: {
        message: {
          id: "server-message-1",
          role: "user",
          content: "Build this",
        },
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as SendAgentChatMessageResult);
    await sendPromise;

    const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
    expect(data?.pages[0]?.messages).toHaveLength(1);
    expect(data?.pages[0]?.messages[0]?.id).toBe("server-message-1");
  });
});

describe("useAgentChatMessages", () => {
  it("keeps optimistic messages when the initial fetch resolves after send starts", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve;
      }) as ReturnType<typeof agentsListAgentChatMessages>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        {
          messages: [
            {
              id: "optimistic-message-1",
              role: "user",
              content: "Build this",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              createdAt: new Date().toISOString(),
            },
          ],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    if (!resolveList) {
      throw new Error("list promise was not created");
    }

    resolveList({
      data: {
        messages: [],
        hasMore: false,
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as ListAgentChatMessagesResult);

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages).toHaveLength(1);
      expect(data?.pages[0]?.messages[0]?.id).toBe("optimistic-message-1");
    });
  });

  it("drops an optimistic message when the initial fetch returns the persisted user message", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve;
      }) as ReturnType<typeof agentsListAgentChatMessages>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        {
          messages: [
            {
              id: "optimistic-message-1",
              role: "user",
              content: "Build this",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              createdAt: new Date().toISOString(),
            },
          ],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    if (!resolveList) {
      throw new Error("list promise was not created");
    }

    resolveList({
      data: {
        messages: [{ id: "server-message-1", role: "user", content: "Build this" }],
        hasMore: false,
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as ListAgentChatMessagesResult);

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages).toHaveLength(1);
      expect(data?.pages[0]?.messages[0]?.id).toBe("server-message-1");
    });
  });

  it("keeps an in-flight image-only send when an earlier persisted image is refetched", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve;
      }) as ReturnType<typeof agentsListAgentChatMessages>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        {
          messages: [
            {
              id: "server-message-1",
              role: "user",
              content: "",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              images: [{ mediaType: "image/png", url: "/img/1" }],
              createdAt: new Date().toISOString(),
            },
            {
              id: "optimistic-message-2",
              role: "user",
              content: "",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              images: [{ mediaType: "image/png", url: "data:image/png;base64,aGVsbG8=" }],
              createdAt: new Date().toISOString(),
            },
          ],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    if (!resolveList) {
      throw new Error("list promise was not created");
    }

    resolveList({
      data: {
        messages: [{ id: "server-message-1", role: "user", content: "", images: [{ mediaType: "MEDIA_TYPE_PNG" }] }],
        hasMore: false,
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as ListAgentChatMessagesResult);

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages.map((message) => message.id)).toEqual([
        "server-message-1",
        "optimistic-message-2",
      ]);
    });
  });

  it("drops an image-only optimistic send when its persisted message is returned", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve;
      }) as ReturnType<typeof agentsListAgentChatMessages>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        {
          messages: [
            {
              id: "optimistic-message-1",
              role: "user",
              content: "",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              images: [{ mediaType: "image/png", url: "data:image/png;base64,aGVsbG8=" }],
              createdAt: new Date().toISOString(),
            },
          ],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    if (!resolveList) {
      throw new Error("list promise was not created");
    }

    resolveList({
      data: {
        messages: [{ id: "server-message-1", role: "user", content: "", images: [{ mediaType: "MEDIA_TYPE_PNG" }] }],
        hasMore: false,
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as ListAgentChatMessagesResult);

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages.map((message) => message.id)).toEqual(["server-message-1"]);
    });
  });

  it("keeps extra optimistic messages when repeated sends share the same content", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve;
      }) as ReturnType<typeof agentsListAgentChatMessages>,
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });

    const makeOptimisticMessage = (id: string) => ({
      id,
      role: "user" as const,
      content: "Build this",
      toolName: "",
      toolCallId: "",
      toolStatus: "",
      createdAt: new Date().toISOString(),
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        {
          messages: [makeOptimisticMessage("optimistic-message-1"), makeOptimisticMessage("optimistic-message-2")],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    if (!resolveList) {
      throw new Error("list promise was not created");
    }

    resolveList({
      data: {
        messages: [{ id: "server-message-1", role: "user", content: "Build this" }],
        hasMore: false,
      },
      request: new Request("https://superplane.test"),
      response: new Response(),
    } as ListAgentChatMessagesResult);

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"));
      expect(data?.pages[0]?.messages.map((message) => message.id)).toEqual([
        "server-message-1",
        "optimistic-message-2",
      ]);
    });
  });
});

describe("useResetCanvasAgentChat", () => {
  it("replaces the canvas chat and clears message caches", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(agentChatKeys.forCanvas("canvas-1"), {
      id: "chat-old",
      canvasId: "canvas-1",
      provider: "anthropic",
      status: "idle",
      createdAt: null,
      updatedAt: null,
    });

    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-old"), {
      pages: [
        {
          messages: [
            { id: "m1", role: "user", content: "hi", toolName: "", toolCallId: "", toolStatus: "", createdAt: null },
          ],
          hasMore: false,
        },
      ],
      pageParams: [""],
    });

    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({
        chat: {
          id: "chat-new",
          canvasId: "canvas-1",
          provider: "anthropic",
          status: "idle",
          createdAt: null,
          updatedAt: null,
        },
      }),
    })) as unknown as typeof fetch;
    vi.stubGlobal("fetch", fetchMock);

    const { result } = renderHook(() => useResetCanvasAgentChat("org-1", "canvas-1"), {
      wrapper: wrapper(queryClient),
    });
    await result.current.mutateAsync();

    const nextChat = queryClient.getQueryData(agentChatKeys.forCanvas("canvas-1")) as any;
    expect(nextChat?.id).toBe("chat-new");
    expect(queryClient.getQueryData(agentChatKeys.messages("chat-old"))).toBeUndefined();

    vi.unstubAllGlobals();
  });
});
