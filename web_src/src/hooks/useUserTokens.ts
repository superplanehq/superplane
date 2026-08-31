import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { meListTokens, meCreateToken, meRevokeToken } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { meKeys } from "@/hooks/useMe";

export const userTokenKeys = {
  all: ["userTokens"] as const,
  list: (organizationId: string) => [...userTokenKeys.all, "list", organizationId] as const,
};

export const useUserTokens = (organizationId: string) => {
  return useQuery({
    queryKey: userTokenKeys.list(organizationId),
    queryFn: async () => {
      const response = await meListTokens(withOrganizationHeader({}));
      return response.data?.tokens || [];
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
    enabled: !!organizationId,
  });
};

export const useCreateUserToken = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { name: string }) => {
      const response = await meCreateToken(
        withOrganizationHeader({
          body: { name: params.name },
        }),
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: userTokenKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId) });
    },
  });
};

export const useRevokeUserToken = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      return meRevokeToken(
        withOrganizationHeader({
          path: { id },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: userTokenKeys.list(organizationId) });
      queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId) });
    },
  });
};
