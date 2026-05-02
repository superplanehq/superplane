import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  integrationsListIntegrations,
  organizationsListIntegrations,
  organizationsDescribeIntegration,
  organizationsListIntegrationResources,
  organizationsCreateIntegration,
  organizationsUpdateIntegration,
  organizationsDeleteIntegration,
  organizationsNextIntegrationSetupStep,
  organizationsPreviousIntegrationSetupStep,
  organizationsUpdateIntegrationSecret,
  organizationsUpdateIntegrationProperty,
  organizationsUpdateIntegrationCapabilities,
} from "@/api-client/sdk.gen";
import type {
  IntegrationCapabilityState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client/types.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { analytics } from "@/lib/analytics";

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
export const useCreateIntegration = (organizationId: string, source: "node_configuration" | "integrations_page") => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      integrationName: string;
      name: string;
      configuration?: Record<string, unknown>;
      capabilities?: string[];
    }) => {
      return await organizationsCreateIntegration(
        withOrganizationHeader({
          path: { id: organizationId },
          body: {
            integrationName: data.integrationName,
            name: data.name,
            configuration: data.configuration,
            capabilities: data.capabilities,
          },
        }),
      );
    },
    onSuccess: (data, variables) => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
      const status = (data.data?.integration?.status?.state || "pending") as "ready" | "error" | "pending";
      analytics.integrationConnectSubmit(variables.integrationName, source, status, organizationId);
    },
  });
};

export const useNextIntegrationSetupStep = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { integrationId: string; inputs?: Record<string, unknown> }) => {
      return await organizationsNextIntegrationSetupStep(
        withOrganizationHeader({
          path: { id: organizationId, integrationId: data.integrationId },
          body: {
            inputs: data.inputs,
          },
        }),
      );
    },
    onSuccess: (response) => {
      const integration = response.data?.integration;
      const integrationId = integration?.metadata?.id;
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });

      if (integrationId) {
        queryClient.setQueryData(integrationKeys.integration(organizationId, integrationId), integration);
      }
    },
  });
};

export const usePreviousIntegrationSetupStep = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { integrationId: string }) => {
      return await organizationsPreviousIntegrationSetupStep(
        withOrganizationHeader({
          path: { id: organizationId, integrationId: data.integrationId },
          body: {},
        }),
      );
    },
    onSuccess: (response) => {
      const integration = response.data?.integration;
      const integrationId = integration?.metadata?.id;

      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });

      if (integrationId) {
        queryClient.setQueryData(integrationKeys.integration(organizationId, integrationId), integration);
      }
    },
  });
};

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

export const useUpdateIntegrationCapabilities = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (capabilities: IntegrationCapabilityState[]) => {
      return await organizationsUpdateIntegrationCapabilities(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
          body: { capabilities },
        }),
      );
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
      const integration = response.data?.integration;
      if (integration) {
        queryClient.setQueryData(integrationKeys.integration(organizationId, integrationId), integration);
        return;
      }

      queryClient.invalidateQueries({
        queryKey: integrationKeys.integration(organizationId, integrationId),
      });
    },
  });
};

function applyIntegrationDescribeCache(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  integrationId: string,
  integration: OrganizationsIntegration | null,
) {
  queryClient.invalidateQueries({
    queryKey: integrationKeys.connected(organizationId),
  });
  if (integration?.metadata?.id) {
    queryClient.setQueryData(integrationKeys.integration(organizationId, integration.metadata.id), integration);
    return;
  }

  queryClient.invalidateQueries({
    queryKey: integrationKeys.integration(organizationId, integrationId),
  });
}

export const useUpdateIntegrationProperty = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (body: { propertyName: string; value: string }) => {
      const response = await organizationsUpdateIntegrationProperty(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
          body: { propertyName: body.propertyName, value: body.value },
        }),
      );
      return response.data?.integration ?? null;
    },
    onSuccess: (integration) => {
      applyIntegrationDescribeCache(queryClient, organizationId, integrationId, integration);
    },
  });
};

export const useUpdateIntegrationSecret = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (body: { secretName: string; value: string }) => {
      const response = await organizationsUpdateIntegrationSecret(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
          body: { secretName: body.secretName, value: body.value },
        }),
      );
      return response.data?.integration ?? null;
    },
    onSuccess: (integration) => {
      applyIntegrationDescribeCache(queryClient, organizationId, integrationId, integration);
    },
  });
};

export const useIntegrationMutations = (organizationId: string, integrationId: string) => {
  const deleteMutation = useDeleteIntegration(organizationId, integrationId);
  const updateCapabilitiesMutation = useUpdateIntegrationCapabilities(organizationId, integrationId);
  const updatePropertyMutation = useUpdateIntegrationProperty(organizationId, integrationId);
  const updateSecretMutation = useUpdateIntegrationSecret(organizationId, integrationId);

  return {
    deleteMutation,
    updateCapabilitiesMutation,
    updatePropertyMutation,
    updateSecretMutation,
    settingsMutationBusy: updatePropertyMutation.isPending || updateSecretMutation.isPending,
  };
};

// Hook to delete an integration
export const useDeleteIntegration = (organizationId: string, integrationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { integrationName: string }) => {
      await organizationsDeleteIntegration(
        withOrganizationHeader({
          path: { id: organizationId, integrationId },
        }),
      );
      return data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: integrationKeys.connected(organizationId),
      });
      queryClient.removeQueries({
        queryKey: integrationKeys.integration(organizationId, integrationId),
      });
      analytics.integrationDelete(variables.integrationName, organizationId);
    },
  });
};
