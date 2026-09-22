import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  groupsAddUserToGroup,
  groupsCreateGroup,
  groupsDeleteGroup,
  groupsRemoveUserFromGroup,
  groupsUpdateGroup,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { organizationKeys } from "./useOrganizationData";

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
