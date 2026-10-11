import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  factoriesListFactoryMcpClients,
  factoriesRevokeFactoryMcpClient,
  factoriesCreateFactoryMcpapiToken,
  factoriesRevokeFactoryMcpapiToken,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export function factoryMCPClientsKey(organizationId: string, factoryId: string) {
  return [...factoryQueryKeys.detail(organizationId, factoryId), "mcp-clients"] as const;
}

export function useFactoryMCPClients(organizationId: string, factoryId: string, enabled = true) {
  return useQuery({
    queryKey: factoryMCPClientsKey(organizationId, factoryId),
    queryFn: async () => {
      const response = await factoriesListFactoryMcpClients(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      return response.data?.clients ?? [];
    },
    enabled: Boolean(organizationId && factoryId) && enabled,
  });
}

export function useCreateFactoryMCPAPIToken(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { name: string; resource: string }) => {
      const response = await factoriesCreateFactoryMcpapiToken(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: { name: input.name, resource: input.resource },
        }),
      );
      return response.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryMCPClientsKey(organizationId, factoryId) });
    },
  });
}

export function useRevokeFactoryMCPClient(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (client: { id: string; kind?: string }) => {
      if (client.kind === "api_token") {
        await factoriesRevokeFactoryMcpapiToken(
          withOrganizationHeader({
            organizationId,
            path: { factoryId, tokenId: client.id },
          }),
        );
        return;
      }
      await factoriesRevokeFactoryMcpClient(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, clientId: client.id },
        }),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryMCPClientsKey(organizationId, factoryId) });
    },
  });
}
