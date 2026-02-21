import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  integrationsListIntegrations,
  organizationsListIntegrations,
  organizationsDescribeIntegration,
  organizationsListIntegrationResources,
  organizationsCreateIntegration,
  organizationsUpdateIntegration,
  organizationsDeleteIntegration,
} from "@/api-client/sdk.gen";
import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";

export const integrationKeys = {
  all: ["integrations"] as const,
  available: () => [...integrationKeys.all, "available"] as const,
  connected: (organizationId: string) => [...integrationKeys.all, "connected", organizationId] as const,
  integration: (organizationId: string, integrationId: string) =>
    [...integrationKeys.connected(organizationId), integrationId] as const,
  resources: (
    organizationId: string,
    integrationId: string,
    resourceType: string,
    parameters?: Record<string, string>,
  ) =>
    [
      ...integrationKeys.integration(organizationId, integrationId),
      "resources",
      resourceType,
      Object.entries(parameters ?? {})
        .map(([k, v]) => `${k}=${v}`)
        .join("&"),
    ] as const,
};

// Hook to fetch available integrations (catalog).
// Normalizes each integration's label (e.g. "github" -> "GitHub") so consumers get correct display names.
export const useAvailableIntegrations = (options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: integrationKeys.available(),
    queryFn: async () => {
      const response = await integrationsListIntegrations(withOrganizationHeader({}));
      const list: IntegrationsIntegrationDefinition[] = response.data?.integrations || [];
      return list.map((integration: IntegrationsIntegrationDefinition) => {
        // Support both camelCase and PascalCase (API may send either)
        const rawLabel = integration.label;
        const rawName = integration.name;
        const displayLabel = getIntegrationTypeDisplayName(rawLabel, rawName) || rawLabel || rawName || "";
        return { ...integration, label: displayLabel } as IntegrationsIntegrationDefinition;
      });
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: options?.enabled ?? true,
  });
};

// Hook to fetch connected integrations for an organization
export const useConnectedIntegrations = (organizationId: string, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: integrationKeys.connected(organizationId),
    queryFn: async () => {
      const response = await organizationsListIntegrations(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data?.integrations || [];
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!organizationId && (options?.enabled ?? true),
  });
};

// Hook to fetch a single integration
export const useIntegration = (organizationId: string, integrationId: string) => {
  return useQuery({
    queryKey: integrationKeys.integration(organizationId, integrationId),
    queryFn: async () => {
      const response = await organizationsDescribeIntegration(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
        }),
      );
      return response.data?.integration || null;
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!organizationId && !!integrationId,
  });
};

export const useIntegrationResources = (
  organizationId: string,
  integrationId: string,
  resourceType: string,
  parameters?: Record<string, string>,
) => {
  return useQuery({
    queryKey: integrationKeys.resources(organizationId, integrationId, resourceType, parameters),
    queryFn: async () => {
      const query: Record<string, string> = {
        type: resourceType,
      };

      for (const [k, v] of Object.entries(parameters ?? {})) {
        query[k] = v;
      }

      const response = await organizationsListIntegrationResources(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
          query,
        }),
      );
      return response.data?.resources || [];
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!organizationId && !!integrationId && !!resourceType,
  });
};

// Hook to create an integration
export const useCreateIntegration = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { integrationName: string; name: string; configuration?: Record<string, unknown> }) => {
      return await organizationsCreateIntegration(
        withOrganizationHeader({
          path: { id: organizationId },
          body: {
            integrationName: data.integrationName,
            name: data.name,
            configuration: data.configuration,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
    },
  });
};

// Hook to update an integration
export const useUpdateIntegration = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { name?: string; configuration?: Record<string, unknown> }) => {
      return await organizationsUpdateIntegration(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
          body: {
            name: data.name,
            configuration: data.configuration,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
      queryClient.invalidateQueries({
        queryKey: integrationKeys.integration(organizationId, integrationId),
      });
    },
  });
};

// Hook to delete an integration
export const useDeleteIntegration = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await organizationsDeleteIntegration(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
      queryClient.removeQueries({
        queryKey: integrationKeys.integration(organizationId, integrationId),
      });
    },
  });
};
