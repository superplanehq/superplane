import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  agentsGetCanvasAgentChat,
  agentsListAgentChatMessages,
  agentsSendAgentChatMessage,
} from "@/api-client/sdk.gen";
import { fromApiChat, fromApiMessage, type AgentChat, type AgentMessage } from "@/components/CanvasToolSidebar/types";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const agentChatKeys = {
  all: ["agentChats"] as const,
  forCanvas: (canvasId: string) => [...agentChatKeys.all, "canvas", canvasId] as const,
  messages: (chatId: string) => [...agentChatKeys.all, "messages", chatId] as const,
};

const PAGE_SIZE = 50;

export function useCanvasAgentChat(
  canvasId: string | undefined,
  organizationId: string | undefined,
  enabled: boolean,
): UseQueryResult<AgentChat | null> {
  return useQuery({
    queryKey: agentChatKeys.forCanvas(canvasId ?? ""),
    enabled: enabled && Boolean(canvasId),
    queryFn: async () => {
      const response = await agentsGetCanvasAgentChat(
        withOrganizationHeader({ organizationId, path: { canvasId: canvasId ?? "" } }),
      );
      return fromApiChat(response.data?.chat);
    },
  });
}

export type AgentMessagesPage = { messages: AgentMessage[]; hasMore: boolean };

export function useAgentChatMessages(chatId: string | null, organizationId: string | undefined, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: agentChatKeys.messages(chatId ?? ""),
    enabled: enabled && Boolean(chatId),
    initialPageParam: "",
    queryFn: async ({ pageParam }): Promise<AgentMessagesPage> => {
      const response = await agentsListAgentChatMessages(
        withOrganizationHeader({
          organizationId,
          path: { chatId: chatId ?? "" },
          query: { beforeId: pageParam || undefined, limit: PAGE_SIZE },
        }),
      );
      const messages = (response.data?.messages ?? []).map(fromApiMessage).filter((m): m is AgentMessage => m !== null);
      return { messages, hasMore: Boolean(response.data?.hasMore) };
    },
    getNextPageParam: (lastPage) => {
      if (!lastPage.hasMore || lastPage.messages.length === 0) return undefined;
      return lastPage.messages[0].id;
    },
  });
}

export function useSendAgentChatMessage(organizationId: string | undefined, _canvasId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ chatId, content }: { chatId: string; content: string }) => {
      const response = await agentsSendAgentChatMessage(
        withOrganizationHeader({ organizationId, path: { chatId }, body: { content } }),
      );
      return fromApiMessage(response.data?.message);
    },
    onSuccess: (data, variables) => {
      if (data) upsertAgentMessageInCache(queryClient, variables.chatId, data);
    },
  });
}

// Mutate the messages cache directly instead of invalidating. Invalidation
// triggers a full server refetch that races with live WS upserts and makes
// tool rows flicker mid-stream.
export function upsertAgentMessageInCache(queryClient: QueryClient, chatId: string, message: AgentMessage): void {
  queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages(chatId), (prev) => {
    if (!prev) return prev;
    const pages = prev.pages.map((p) => ({ ...p, messages: p.messages.slice() }));
    for (const page of pages) {
      const idx = page.messages.findIndex((m) => m.id === message.id);
      if (idx !== -1) {
        page.messages[idx] = message;
        return { ...prev, pages };
      }
    }
    if (pages.length === 0) {
      return { ...prev, pages: [{ messages: [message], hasMore: false }] };
    }
    pages[0].messages.push(message);
    return { ...prev, pages };
  });
}
