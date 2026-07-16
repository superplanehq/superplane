import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  apiKeysListApiKeys,
  apiKeysCreateApiKey,
  apiKeysDescribeApiKey,
  apiKeysUpdateApiKey,
  apiKeysDeleteApiKey,
  apiKeysRegenerateApiKeyToken,
} from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const apiKeyKeys = {
  all: ["apiKeys"] as const,
  list: (orgId: string) => [...apiKeyKeys.all, "list", orgId] as const,
  detail: (orgId: string, id: string) => [...apiKeyKeys.all, "detail", orgId, id] as const,
};

export const useAPIKeys = (organizationId: string) => {
  return useQuery({
    queryKey: apiKeyKeys.list(organizationId),
    queryFn: async () => {
      const response = await apiKeysListApiKeys(withOrganizationHeader({}));
      return response.data?.apiKeys || [];
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
    enabled: !!organizationId,
  });
};

export const useAPIKey = (organizationId: string, id: string) => {
  return useQuery({
    queryKey: apiKeyKeys.detail(organizationId, id),
    queryFn: async () => {
      const response = await apiKeysDescribeApiKey(
        withOrganizationHeader({
          path: { id },
        }),
      );
      return response.data?.apiKey || null;
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
    enabled: !!organizationId && !!id,
  });
};

export const useCreateAPIKey = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      name: string;
      description: string;
      role: string;
      expiresAt?: string;
      canvasIds: string[];
    }) => {
      return apiKeysCreateApiKey(
        withOrganizationHeader({
          body: {
            name: params.name,
            description: params.description,
            role: params.role,
            expiresAt: params.expiresAt,
            canvasIds: params.canvasIds,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list(organizationId) });
    },
  });
};

export const useUpdateAPIKey = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      id: string;
      name: string;
      description: string;
      expiresAt?: string;
      clearExpiresAt: boolean;
      canvasIds?: string[];
    }) => {
      return apiKeysUpdateApiKey(
        withOrganizationHeader({
          path: { id: params.id },
          body: {
            name: params.name,
            description: params.description,
            expiresAt: params.expiresAt,
            clearExpiresAt: params.clearExpiresAt,
            ...(params.canvasIds !== undefined ? { canvasIds: params.canvasIds } : {}),
          },
        }),
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.detail(organizationId, variables.id) });
    },
  });
};

export const useDeleteAPIKey = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiKeysDeleteApiKey(
        withOrganizationHeader({
          path: { id },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.list(organizationId) });
    },
  });
};

export const useRegenerateAPIKeyToken = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return apiKeysRegenerateApiKeyToken(
        withOrganizationHeader({
          path: { id },
          body: {},
        }),
      );
    },
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: apiKeyKeys.detail(organizationId, id) });
    },
  });
};
