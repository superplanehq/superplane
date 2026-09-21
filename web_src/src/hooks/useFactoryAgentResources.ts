import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  factoriesCreateFactoryAgentResource,
  factoriesDeleteFactoryAgentResource,
  factoriesDisconnectFactoryAgentResourceOAuth,
  factoriesListFactoryAgentResources,
  factoriesStartFactoryAgentResourceOAuth,
  factoriesUpdateFactoryAgentResource,
  type FactoriesCreateFactoryAgentResourceBody,
  type FactoriesFactoryAgentResourceKind,
  type FactoriesUpdateFactoryAgentResourceBody,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export function factoryAgentResourcesKey(
  organizationId: string,
  factoryId: string,
  kind?: FactoriesFactoryAgentResourceKind,
) {
  return [...factoryQueryKeys.detail(organizationId, factoryId), "agent-resources", kind ?? "all"] as const;
}

function invalidateAgentResources(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
) {
  void queryClient.invalidateQueries({
    queryKey: [...factoryQueryKeys.detail(organizationId, factoryId), "agent-resources"],
  });
}

export function useFactoryAgentResources(
  organizationId: string,
  factoryId: string,
  kind: FactoriesFactoryAgentResourceKind,
  enabled = true,
) {
  return useQuery({
    queryKey: factoryAgentResourcesKey(organizationId, factoryId, kind),
    queryFn: async () => {
      const response = await factoriesListFactoryAgentResources(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { kind },
        }),
      );
      return response.data?.resources ?? [];
    },
    enabled: Boolean(organizationId && factoryId) && enabled,
  });
}

export function useCreateFactoryAgentResource(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: FactoriesCreateFactoryAgentResourceBody) => {
      const response = await factoriesCreateFactoryAgentResource(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body,
        }),
      );
      if (!response.data?.resource) {
        throw new Error("Failed to create agent resource");
      }
      return response.data.resource;
    },
    onSuccess: () => invalidateAgentResources(queryClient, organizationId, factoryId),
  });
}

export function useUpdateFactoryAgentResource(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { resourceId: string } & FactoriesUpdateFactoryAgentResourceBody) => {
      const { resourceId, ...body } = input;
      const response = await factoriesUpdateFactoryAgentResource(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, resourceId },
          body,
        }),
      );
      if (!response.data?.resource) {
        throw new Error("Failed to update agent resource");
      }
      return response.data.resource;
    },
    onSuccess: () => invalidateAgentResources(queryClient, organizationId, factoryId),
  });
}

export function useDeleteFactoryAgentResource(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (resourceId: string) => {
      await factoriesDeleteFactoryAgentResource(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, resourceId },
        }),
      );
      return resourceId;
    },
    onSuccess: () => invalidateAgentResources(queryClient, organizationId, factoryId),
  });
}

export function useStartFactoryAgentResourceOAuth(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (resourceId: string) => {
      const response = await factoriesStartFactoryAgentResourceOAuth(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, resourceId },
          body: {},
        }),
      );
      return response.data ?? {};
    },
    onSuccess: () => invalidateAgentResources(queryClient, organizationId, factoryId),
  });
}

export function useDisconnectFactoryAgentResourceOAuth(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (resourceId: string) => {
      const response = await factoriesDisconnectFactoryAgentResourceOAuth(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, resourceId },
          body: {},
        }),
      );
      if (!response.data?.resource) {
        throw new Error("Failed to disconnect agent resource");
      }
      return response.data.resource;
    },
    onSuccess: () => invalidateAgentResources(queryClient, organizationId, factoryId),
  });
}
