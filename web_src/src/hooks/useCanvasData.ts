import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
  integrationsListIntegrations,
  superplaneAddUser,
  superplaneRemoveUser,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../utils/withOrganizationHeader";
import type { RolesAssignRoleData } from "../api-client/types.gen";

export const canvasKeys = {
  all: ["canvas"] as const,
  details: (canvasId: string) => [...canvasKeys.all, "details", canvasId] as const,
  users: (canvasId: string) => [...canvasKeys.all, "users", canvasId] as const,
  roles: (canvasId: string) => [...canvasKeys.all, "roles", canvasId] as const,
  integrations: (canvasId?: string) =>
    canvasId ? ([...canvasKeys.all, "integrations", canvasId] as const) : (["integrations"] as const),
};

export const useCanvasRoles = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.roles(canvasId),
    queryFn: async () => {
      const response = await rolesListRoles(
        withOrganizationHeader({
          query: { domainId: canvasId, domainType: "DOMAIN_TYPE_CANVAS" },
        }),
      );
      return response.data?.roles || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId,
  });
};

export const useCanvasUsers = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.users(canvasId),
    queryFn: async () => {
      const response = await usersListUsers(
        withOrganizationHeader({
          query: { domainId: canvasId, domainType: "DOMAIN_TYPE_CANVAS" },
        }),
      );
      return response.data?.users || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId,
  });
};

export const useOrganizationUsersForCanvas = (organizationId: string) => {
  return useQuery({
    queryKey: ["organizationUsers", organizationId],
    queryFn: async () => {
      const response = await usersListUsers(
        withOrganizationHeader({
          query: { domainId: organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
        }),
      );
      return response.data?.users || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId,
  });
};

export const useAssignCanvasRole = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { userId?: string; role: string }) => {
      return await rolesAssignRole(
        withOrganizationHeader({
          path: { roleName: params.role },
          body: {
            userId: params.userId,
            domainId: canvasId,
            domainType: "DOMAIN_TYPE_CANVAS",
          },
        } as RolesAssignRoleData),
      );
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) });
      // Invalidate organization invitations - we need to get the organizationId
      queryClient.invalidateQueries({ queryKey: ["organization", "invitations"] });
    },
  });
};

export const useAddCanvasUser = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { userId: string }) => {
      return await superplaneAddUser(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          body: { userId: params.userId },
        }),
      );
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) });
    },
  });
};

export const useRemoveCanvasUser = (canvasId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { userId: string }) => {
      return await superplaneRemoveUser(
        withOrganizationHeader({
          path: {
            canvasIdOrName: canvasId,
            userId: params.userId,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) });
      // Invalidate organization invitations
      queryClient.invalidateQueries({ queryKey: ["organization", "invitations"] });
    },
  });
};

// Integration-related hooks
export const useIntegrations = () => {
  return useQuery({
    queryKey: canvasKeys.integrations(),
    queryFn: async () => {
      const response = await integrationsListIntegrations(withOrganizationHeader());
      return response.data?.integrations || [];
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};
