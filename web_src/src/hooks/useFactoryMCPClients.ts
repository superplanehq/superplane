import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoriesListFactoryMcpClients, factoriesRevokeFactoryMcpClient } from "@/api-client";
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

export function useRevokeFactoryMCPClient(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (clientId: string) => {
      await factoriesRevokeFactoryMcpClient(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, clientId },
        }),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryMCPClientsKey(organizationId, factoryId) });
    },
  });
}
