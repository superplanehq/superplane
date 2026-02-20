import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  serviceAccountsListServiceAccounts,
  serviceAccountsCreateServiceAccount,
  serviceAccountsDescribeServiceAccount,
  serviceAccountsUpdateServiceAccount,
  serviceAccountsDeleteServiceAccount,
  serviceAccountsRegenerateServiceAccountToken,
} from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";

export const serviceAccountKeys = {
  all: ["serviceAccounts"] as const,
  list: (orgId: string) => [...serviceAccountKeys.all, "list", orgId] as const,
  detail: (orgId: string, id: string) => [...serviceAccountKeys.all, "detail", orgId, id] as const,
};

export const useServiceAccounts = (organizationId: string) => {
  return useQuery({
    queryKey: serviceAccountKeys.list(organizationId),
    queryFn: async () => {
      const response = await serviceAccountsListServiceAccounts(withOrganizationHeader({}));
      return response.data?.serviceAccounts || [];
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
    enabled: !!organizationId,
  });
};

export const useServiceAccount = (organizationId: string, id: string) => {
  return useQuery({
    queryKey: serviceAccountKeys.detail(organizationId, id),
    queryFn: async () => {
      const response = await serviceAccountsDescribeServiceAccount(
        withOrganizationHeader({
          path: { id },
        }),
      );
      return response.data?.serviceAccount || null;
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
    enabled: !!organizationId && !!id,
  });
};

export const useCreateServiceAccount = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { name: string; description: string; role: string }) => {
      return serviceAccountsCreateServiceAccount(
        withOrganizationHeader({
          body: {
            name: params.name,
            description: params.description,
            role: params.role,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: serviceAccountKeys.list(organizationId) });
    },
  });
};

export const useUpdateServiceAccount = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { id: string; name: string; description: string }) => {
      return serviceAccountsUpdateServiceAccount(
        withOrganizationHeader({
          path: { id: params.id },
          body: {
            name: params.name,
            description: params.description,
          },
        }),
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: serviceAccountKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: serviceAccountKeys.detail(organizationId, variables.id) });
    },
  });
};

export const useDeleteServiceAccount = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return serviceAccountsDeleteServiceAccount(
        withOrganizationHeader({
          path: { id },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: serviceAccountKeys.list(organizationId) });
    },
  });
};

export const useRegenerateServiceAccountToken = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return serviceAccountsRegenerateServiceAccountToken(
        withOrganizationHeader({
          path: { id },
          body: {},
        }),
      );
    },
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: serviceAccountKeys.detail(organizationId, id) });
    },
  });
};
