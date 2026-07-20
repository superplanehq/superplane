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
  agentsResetCanvasAgentChat,
  agentsSendAgentChatMessage,
} from "@/api-client/sdk.gen";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import {
  fromApiChat,
  fromApiMessage,
  apiImageMediaTypeToMime,
  type AgentChat,
  type AgentMessage,
  type AgentOutgoingImage,
} from "@/components/CanvasToolSidebar/types";
import { analytics } from "@/lib/analytics";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const agentChatKeys = {
  all: ["agentChats"] as const,
  forCanvas: (canvasId: string) => [...agentChatKeys.all, "canvas", canvasId] as const,
  messages: (chatId: string) => [...agentChatKeys.all, "messages", chatId] as const,
};

const PAGE_SIZE = 50;
const agentModeToApiMode = {
  builder: "MODE_BUILDER",
  operator: "MODE_OPERATOR",
} as const;

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
  const queryClient = useQueryClient();

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
      const messages = (response.data?.messages ?? [])
        .map((message) => fromApiMessage(message, chatId ?? "", organizationId))
        .filter((m): m is AgentMessage => m !== null);
      if (!pageParam && chatId) {
        return {
          messages: mergePendingOptimisticMessages(
            messages,
            queryClient.getQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages(chatId)),
          ),
          hasMore: Boolean(response.data?.hasMore),
        };
      }

      return { messages, hasMore: Boolean(response.data?.hasMore) };
    },
    getNextPageParam: (lastPage) => {
      if (!lastPage.hasMore || lastPage.messages.length === 0) return undefined;
      return lastPage.messages[0].id;
    },
  });
}

export function useSendAgentChatMessage(organizationId: string | undefined, canvasId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      chatId,
      content,
      mode,
      images,
      autoLayoutOnUpdateEnabled,
    }: {
      chatId: string;
      content: string;
      mode?: AgentMode;
      images?: AgentOutgoingImage[];
      autoLayoutOnUpdateEnabled?: boolean;
    }) => {
      const response = await agentsSendAgentChatMessage(
        withOrganizationHeader({
          organizationId,
          path: { chatId },
          body: {
            content,
            mode: mode ? agentModeToApiMode[mode] : undefined,
            autoLayoutOnUpdateEnabled,
            images: images && images.length > 0 ? images : undefined,
          },
        }),
      );
      return fromApiMessage(response.data?.message, chatId, organizationId);
    },
    onMutate: ({ chatId, content, mode, images }) => {
      const submittedAt = Date.now();
      const optimisticMessage: AgentMessage = {
        id: `optimistic-${Date.now()}-${Math.random().toString(36).slice(2)}`,
        role: "user",
        content,
        toolName: "",
        toolCallId: "",
        toolStatus: "",
        images: images?.map(({ mediaType, data }) => {
          const mimeType = apiImageMediaTypeToMime(mediaType);
          return { mediaType: mimeType, url: `data:${mimeType};base64,${data}` };
        }),
        createdAt: new Date().toISOString(),
      };
      upsertAgentMessageInCache(queryClient, chatId, optimisticMessage);
      analytics.agentMessageSendSubmitted(chatId, canvasId, organizationId, mode);
      return { mode, optimisticMessageId: optimisticMessage.id, submittedAt };
    },
    onSuccess: (data, variables, context) => {
      if (context?.optimisticMessageId) {
        removeAgentMessageFromCache(queryClient, variables.chatId, context.optimisticMessageId);
      }
      if (context?.submittedAt) {
        analytics.agentMessageSendAcknowledged(
          variables.chatId,
          canvasId,
          organizationId,
          context.mode,
          Date.now() - context.submittedAt,
        );
      }
      if (data) upsertAgentMessageInCache(queryClient, variables.chatId, data);
    },
    onError: (_error, variables, context) => {
      if (context?.optimisticMessageId) {
        removeAgentMessageFromCache(queryClient, variables.chatId, context.optimisticMessageId);
      }
      if (context?.submittedAt) {
        analytics.agentMessageSendFailed(
          variables.chatId,
          canvasId,
          organizationId,
          context.mode,
          Date.now() - context.submittedAt,
        );
      }
    },
  });
}

function mergePendingOptimisticMessages(
  messages: AgentMessage[],
  currentData: InfiniteData<AgentMessagesPage> | undefined,
): AgentMessage[] {
  const cachedMessages = currentData?.pages.flatMap((page) => page.messages) ?? [];
  const optimisticMessages = cachedMessages.filter(isOptimisticAgentMessage);
  if (optimisticMessages.length === 0) {
    return messages;
  }

  const knownMessageIds = new Set(cachedMessages.map((message) => message.id));
  const serverMessageIds = new Set(messages.map((message) => message.id));

  const newlyPersistedCounts = new Map<string, number>();
  for (const message of messages) {
    if (message.role !== "user" || isOptimisticAgentMessage(message) || knownMessageIds.has(message.id)) {
      continue;
    }

    const key = optimisticMessageMatchKey(message);
    newlyPersistedCounts.set(key, (newlyPersistedCounts.get(key) ?? 0) + 1);
  }

  const pendingMessages = optimisticMessages.filter((message) => {
    if (serverMessageIds.has(message.id)) {
      return false;
    }

    const key = optimisticMessageMatchKey(message);
    const persistedCount = newlyPersistedCounts.get(key) ?? 0;
    if (persistedCount === 0) {
      return true;
    }

    newlyPersistedCounts.set(key, persistedCount - 1);
    return false;
  });

  return [...messages, ...pendingMessages];
}

function isOptimisticAgentMessage(message: AgentMessage): boolean {
  return message.id.startsWith("optimistic-");
}

function optimisticMessageMatchKey(message: AgentMessage): string {
  return `${message.role}:${message.content}`;
}

export function useInterruptAgentChat(organizationId: string | undefined) {
  return useMutation({
    mutationFn: async ({ chatId }: { chatId: string }) => {
      const res = await fetch(`/api/v1/agents/chats/${chatId}/interrupt`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "x-organization-id": organizationId ?? "" },
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Interrupt failed: ${res.status}`);
    },
  });
}

export function useResetCanvasAgentChat(organizationId: string | undefined, canvasId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      if (!canvasId) throw new Error("Canvas is required");
      if (!organizationId) throw new Error("Organization is required");

      const response = await agentsResetCanvasAgentChat(
        withOrganizationHeader({
          organizationId,
          path: { canvasId },
          body: {},
          headers: { "Content-Type": "application/json" },
        }),
      );
      return fromApiChat(response.data?.chat);
    },
    onSuccess: (nextChat) => {
      if (!canvasId) return;
      const previousChat = queryClient.getQueryData<AgentChat | null>(agentChatKeys.forCanvas(canvasId));
      queryClient.setQueryData(agentChatKeys.forCanvas(canvasId), nextChat);

      if (previousChat?.id) queryClient.removeQueries({ queryKey: agentChatKeys.messages(previousChat.id) });
      if (nextChat?.id) queryClient.removeQueries({ queryKey: agentChatKeys.messages(nextChat.id) });
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

function removeAgentMessageFromCache(queryClient: QueryClient, chatId: string, messageId: string): void {
  queryClient.setQueryData<InfiniteData<AgentMessagesPage>>(agentChatKeys.messages(chatId), (prev) => {
    if (!prev) return prev;

    return {
      ...prev,
      pages: prev.pages.map((page) => ({
        ...page,
        messages: page.messages.filter((message) => message.id !== messageId),
      })),
    };
  });
}

export function useDefineAgentOutcome(organizationId: string | undefined) {
  return useMutation({
    mutationFn: async ({
      chatId,
      description,
      rubric,
      maxIterations,
    }: {
      chatId: string;
      description: string;
      rubric: string;
      maxIterations?: number;
    }) => {
      const res = await fetch(`/api/v1/agents/chats/${chatId}/outcome`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "x-organization-id": organizationId ?? "" },
        credentials: "include",
        body: JSON.stringify({ chat_id: chatId, description, rubric, max_iterations: maxIterations || 3 }),
      });
      if (!res.ok) throw new Error(`Define outcome failed: ${res.status}`);
    },
  });
}
