import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { agentsListAgentChatMessages } from "@/api-client/sdk.gen";
import type { AgentMessagesPage } from "./useAgentChats";
import { agentChatKeys, upsertAgentMessageInCache, useAgentChatMessages } from "./useAgentChats";

vi.mock("@/api-client/sdk.gen", () => ({
  agentsListAgentChatMessages: vi.fn(),
  agentsResetCanvasAgentChat: vi.fn(),
  agentsSendAgentChatMessage: vi.fn(),
}));

function wrapper(queryClient: QueryClient) {
  return function TestQueryClientProvider({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useAgentChatMessages live reconciliation", () => {
  it.each(["assistant", "tool", "unchanged", "displaced"])(
    "reconciles history with concurrent %s messages",
    async (scenario) => {
      type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
      let resolveList: ((value: ListAgentChatMessagesResult) => void) | undefined;
      vi.mocked(agentsListAgentChatMessages).mockReturnValue(
        new Promise((resolve) => {
          resolveList = resolve;
        }) as ReturnType<typeof agentsListAgentChatMessages>,
      );
      const tool = {
        id: "tool-1",
        role: "tool",
        content: "Inspecting the canvas",
        toolName: "bash",
        toolCallId: "call-1",
        toolStatus: "started",
        createdAt: new Date().toISOString(),
      };
      const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
        pages: [{ messages: scenario === "assistant" ? [] : [tool], hasMore: false }],
        pageParams: [""],
      });
      const { result } = renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
        wrapper: wrapper(queryClient),
      });
      await waitFor(() => expect(result.current.isFetching).toBe(true));
      const received =
        scenario === "assistant"
          ? { ...tool, id: "reply-1", role: "assistant", content: "The suggestion is complete.", toolStatus: "" }
          : { ...tool, toolStatus: "finished" };
      if (scenario !== "unchanged") upsertAgentMessageInCache(queryClient, "chat-1", received);
      if (!resolveList) throw new Error("list promise was not created");
      const newer = { ...tool, id: "newer-tool" };
      resolveList({
        data: { messages: scenario === "tool" ? [tool] : scenario === "displaced" ? [newer] : [], hasMore: false },
        request: new Request("https://superplane.test"),
        response: new Response(),
      } as ListAgentChatMessagesResult);

      await waitFor(() => expect(result.current.isFetching).toBe(false));
      expect(result.current.data?.pages[0].messages).toEqual(
        scenario === "unchanged" ? [] : scenario === "displaced" ? [{ ...newer, images: [] }] : [received],
      );
    },
  );

  it("keeps live messages received while an older history page refreshes", async () => {
    type ListAgentChatMessagesResult = Awaited<ReturnType<typeof agentsListAgentChatMessages>>;
    const toListResult = (messages: unknown[], hasMore: boolean) =>
      ({
        data: { messages, hasMore },
        request: new Request("https://superplane.test"),
        response: new Response(),
      }) as ListAgentChatMessagesResult;
    const tool = {
      id: "tool-1",
      role: "tool",
      content: "Inspecting the canvas",
      toolName: "bash",
      toolCallId: "call-1",
      toolStatus: "started",
      createdAt: new Date().toISOString(),
    };
    const older = { ...tool, id: "older-1", toolStatus: "finished" };
    let resolveOlderPage: ((value: ListAgentChatMessagesResult) => void) | undefined;
    vi.mocked(agentsListAgentChatMessages).mockImplementation((async ({ query }: { query?: { beforeId?: string } }) => {
      if (!query?.beforeId) return toListResult([tool], true);
      return new Promise<ListAgentChatMessagesResult>((resolve) => {
        resolveOlderPage = resolve;
      });
    }) as unknown as typeof agentsListAgentChatMessages);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"), {
      pages: [
        { messages: [tool], hasMore: true },
        { messages: [older], hasMore: false },
      ],
      pageParams: ["", "tool-1"],
    });
    const { result } = renderHook(() => useAgentChatMessages("chat-1", "org-1", true), {
      wrapper: wrapper(queryClient),
    });
    await waitFor(() => expect(resolveOlderPage).toBeDefined());

    const finishedTool = { ...tool, toolStatus: "finished" };
    const reply = { ...tool, id: "reply-1", role: "assistant", content: "Done.", toolStatus: "" };
    upsertAgentMessageInCache(queryClient, "chat-1", finishedTool);
    upsertAgentMessageInCache(queryClient, "chat-1", reply);
    resolveOlderPage?.(toListResult([], false));

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(result.current.data?.pages.flatMap((page) => page.messages)).toEqual(
      expect.arrayContaining([finishedTool, reply]),
    );
    expect(
      queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages("chat-1"))?.pages[0].messages,
    ).toEqual(expect.arrayContaining([finishedTool, reply]));
  });
});
