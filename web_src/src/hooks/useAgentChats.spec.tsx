import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { agentsSendAgentChatMessage } from "@/api-client/sdk.gen";
import type { AgentMessagesPage } from "./useAgentChats";
import { agentChatKeys, useSendAgentChatMessage } from "./useAgentChats";

vi.mock("@/api-client/sdk.gen", () => ({
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
