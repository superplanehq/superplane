import { useMutation, useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import {
  agentsCreateAgentChat,
  agentsDeleteAgentChat,
  agentsListAgentChatMessages,
  agentsListAgentChats,
  agentsSendAgentChatMessage,
} from "@/api-client/sdk.gen";
import { fromApiChat, fromApiMessage, type AgentChat, type AgentMessage } from "@/components/AgentSidebar/types";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const agentChatKeys = {
  all: ["agentChats"] as const,
  list: (canvasId: string) => [...agentChatKeys.all, "list", canvasId] as const,
  messages: (chatId: string) => [...agentChatKeys.all, "messages", chatId] as const,
};

export function useAgentChats(
  canvasId: string | undefined,
  organizationId: string | undefined,
  enabled: boolean,
): UseQueryResult<AgentChat[]> {
  return useQuery({
    queryKey: agentChatKeys.list(canvasId ?? ""),
    enabled: enabled && Boolean(canvasId),
    queryFn: async () => {
      const response = await agentsListAgentChats(withOrganizationHeader({ organizationId, query: { canvasId } }));
      const chats = response.data?.chats ?? [];
      return chats.map(fromApiChat).filter((c): c is AgentChat => c !== null);
    },
  });
}

export function useAgentChatMessages(
  chatId: string | null,
  organizationId: string | undefined,
  enabled: boolean,
): UseQueryResult<AgentMessage[]> {
  return useQuery({
    queryKey: agentChatKeys.messages(chatId ?? ""),
    enabled: enabled && Boolean(chatId),
    queryFn: async () => {
      const response = await agentsListAgentChatMessages(
        withOrganizationHeader({ organizationId, path: { chatId: chatId ?? "" } }),
      );
      const messages = response.data?.messages ?? [];
      return messages.map(fromApiMessage).filter((m): m is AgentMessage => m !== null);
    },
  });
}

export function useCreateAgentChat(canvasId: string | undefined, organizationId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const response = await agentsCreateAgentChat(withOrganizationHeader({ organizationId, body: { canvasId } }));
      return fromApiChat(response.data?.chat);
    },
    onSuccess: () => {
      if (canvasId) {
        void queryClient.invalidateQueries({ queryKey: agentChatKeys.list(canvasId) });
      }
    },
  });
}

export function useSendAgentChatMessage(organizationId: string | undefined, canvasId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ chatId, content }: { chatId: string; content: string }) => {
      const response = await agentsSendAgentChatMessage(
        withOrganizationHeader({ organizationId, path: { chatId }, body: { content } }),
      );
      return fromApiMessage(response.data?.message);
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: agentChatKeys.messages(variables.chatId) });
      // First message sets the title server-side; refresh the list.
      if (canvasId) {
        void queryClient.invalidateQueries({ queryKey: agentChatKeys.list(canvasId) });
      }
    },
  });
}

export function useArchiveAgentChat(canvasId: string | undefined, organizationId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (chatId: string) => {
      await agentsDeleteAgentChat(withOrganizationHeader({ organizationId, path: { chatId } }));
    },
    onSuccess: () => {
      if (canvasId) {
        void queryClient.invalidateQueries({ queryKey: agentChatKeys.list(canvasId) });
      }
    },
  });
}
