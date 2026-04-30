import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { AuthorizationDomainType, OrganizationsRemoveUserData, RolesCreateRoleRequest } from "@/api-client";
import {
  groupsAddUserToGroup,
  groupsCreateGroup,
  groupsDeleteGroup,
  groupsRemoveUserFromGroup,
  groupsUpdateGroup,
  organizationsDeleteOrganization,
  organizationsRemoveUser,
  organizationsResetInviteLink,
  organizationsUpdateInviteLink,
  organizationsUpdateOrganization,
  rolesAssignRole,
  rolesCreateRole,
  rolesDeleteRole,
  rolesUpdateRole,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { canvasKeys } from "./useCanvasData";
import { organizationKeys } from "./organizationQueryKeys";

export const useDeleteOrganization = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await organizationsDeleteOrganization(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.all });
    },
  });
};

export const useAssignRole = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { userId?: string; userEmail?: string; roleName: string }) => {
      return await rolesAssignRole(
        withOrganizationHeader({
          path: {
            roleName: params.roleName,
          },
          body: {
            userId: params.userId,
            userEmail: params.userEmail,
            domainType: "DOMAIN_TYPE_ORGANIZATION",
            domainId: organizationId,
          },
        }),
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) });
      if (variables.userId) {
        queryClient.invalidateQueries({ queryKey: ["permissions", organizationId, variables.userId] });
        return;
      }
      queryClient.invalidateQueries({ queryKey: ["permissions", organizationId] });
    },
  });
};

export const useRemoveOrganizationSubject = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { userId: string }) => {
      return await organizationsRemoveUser(
        withOrganizationHeader({
          path: {
            id: organizationId,
            userId: params.userId,
          },
        } as OrganizationsRemoveUserData),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) });
    },
  });
};

export const useUpdateOrganizationInviteLink = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (enabled: boolean) => {
      const response = await organizationsUpdateInviteLink(
        withOrganizationHeader({
          path: { id: organizationId },
          body: { enabled },
        }),
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.inviteLink(organizationId) });
    },
  });
};

export const useResetOrganizationInviteLink = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await organizationsResetInviteLink(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.inviteLink(organizationId) });
    },
  });
};

export const useCreateGroup = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      organizationId: string;
      groupName: string;
      displayName?: string;
      description?: string;
      role?: string;
    }) => {
      return await groupsCreateGroup(
        withOrganizationHeader({
          body: {
            group: {
              metadata: {
                name: params.groupName,
              },
              spec: {
                displayName: params.displayName,
                description: params.description,
                role: params.role,
              },
            },
            domainId: params.organizationId,
            domainType: "DOMAIN_TYPE_ORGANIZATION",
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) });
    },
  });
};

export const useUpdateGroup = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      groupName: string;
      organizationId: string;
      displayName?: string;
      description?: string;
      role?: string;
    }) => {
      return await groupsUpdateGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          body: {
            group: {
              metadata: {
                name: params.groupName,
              },
              spec: {
                displayName: params.displayName,
                description: params.description,
                role: params.role,
              },
            },
            domainId: params.organizationId,
            domainType: "DOMAIN_TYPE_ORGANIZATION",
          },
        }),
      );
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) });
      queryClient.invalidateQueries({ queryKey: organizationKeys.group(organizationId, variables.groupName) });
    },
  });
};

export const useDeleteGroup = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { groupName: string; organizationId: string }) => {
      return await groupsDeleteGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          query: { domainId: params.organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) });
    },
  });
};

export const useAddUserToGroup = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { groupName: string; userId?: string; userEmail?: string; organizationId: string }) => {
      return await groupsAddUserToGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          body: {
            userId: params.userId,
            userEmail: params.userEmail,
            domainId: params.organizationId,
            domainType: "DOMAIN_TYPE_ORGANIZATION",
          },
        }),
      );
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(organizationId, variables.groupName) });
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) });
    },
  });
};

export const useRemoveUserFromGroup = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { groupName: string; userId: string; organizationId: string }) => {
      return await groupsRemoveUserFromGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          body: {
            userId: params.userId,
            domainId: params.organizationId,
            domainType: "DOMAIN_TYPE_ORGANIZATION",
          },
        }),
      );
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(organizationId, variables.groupName) });
    },
  });
};

export const useCreateRole = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: RolesCreateRoleRequest) => {
      return await rolesCreateRole(
        withOrganizationHeader({
          body: params,
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) });
    },
  });
};

export const useUpdateRole = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      roleName: string;
      domainType: AuthorizationDomainType | undefined;
      domainId: string;
      permissions: Array<{ resource: string; action: string; domainType: AuthorizationDomainType | undefined }>;
      displayName?: string;
      description?: string;
    }) => {
      return await rolesUpdateRole(
        withOrganizationHeader({
          path: { roleName: params.roleName },
          body: {
            domainType: params.domainType,
            domainId: params.domainId,
            role: {
              metadata: {
                name: params.roleName,
              },
              spec: {
                permissions: params.permissions,
                displayName: params.displayName,
                description: params.description,
              },
            },
          },
        }),
      );
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) });
      queryClient.invalidateQueries({ queryKey: organizationKeys.role(organizationId, variables.roleName) });
    },
  });
};

export const useDeleteRole = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { roleName: string; domainType: AuthorizationDomainType; domainId: string }) => {
      return await rolesDeleteRole(
        withOrganizationHeader({
          path: { roleName: params.roleName },
          query: {
            domainType: params.domainType,
            domainId: params.domainId,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) });
    },
  });
};

export const useUpdateOrganization = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      name?: string;
      description?: string;
      changeManagementEnabled?: boolean;
      /** When set, replaces stored OAuth allow-list for invitations (empty = no restriction). */
      allowedOauthProviders?: string[];
    }) => {
      const hasSpecUpdate =
        typeof params.changeManagementEnabled === "boolean" || params.allowedOauthProviders !== undefined;

      const spec = hasSpecUpdate
        ? {
            ...(typeof params.changeManagementEnabled === "boolean"
              ? { changeManagementEnabled: params.changeManagementEnabled }
              : {}),
            ...(params.allowedOauthProviders !== undefined
              ? { allowedOauthProviders: { providers: params.allowedOauthProviders } }
              : {}),
          }
        : undefined;

      return await organizationsUpdateOrganization(
        withOrganizationHeader({
          path: { id: organizationId },
          body: {
            organization: {
              metadata: {
                name: params.name,
                description: params.description,
              },
              spec,
            },
          },
        }),
      );
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.details(organizationId) });
      if (typeof variables.changeManagementEnabled === "boolean") {
        queryClient.invalidateQueries({ queryKey: canvasKeys.all });
      }
    },
  });
};
