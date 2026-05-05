import { useQuery } from "@tanstack/react-query";
import {
  usersListUsers,
  rolesListRoles,
  groupsListGroups,
  groupsDescribeGroup,
  groupsListGroupUsers,
  rolesDescribeRole,
  organizationsDescribeOrganization,
  organizationsGetInviteLink,
  organizationsGetAgentSettings,
  organizationsDescribeUsage,
} from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { organizationKeys } from "./organizationQueryKeys";

export { organizationKeys };
export * from "./useOrganizationMutations";

// Hooks for fetching data
export const useOrganization = (organizationId: string, enabled = true) => {
  return useQuery({
    queryKey: organizationKeys.details(organizationId),
    queryFn: async () => {
      const response = await organizationsDescribeOrganization(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data?.organization || null;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId && enabled,
  });
};

export const useOrganizationUsers = (organizationId: string, includeRoles = false) => {
  return useQuery({
    queryKey: includeRoles
      ? [...organizationKeys.users(organizationId), includeRoles]
      : organizationKeys.users(organizationId),
    queryFn: async () => {
      const response = await usersListUsers(
        withOrganizationHeader({
          query: {
            domainType: "DOMAIN_TYPE_ORGANIZATION",
            domainId: organizationId,
            includeRoles,
          },
        }),
      );
      return response.data?.users || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useOrganizationRoles = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.roles(organizationId),
    queryFn: async () => {
      const response = await rolesListRoles(
        withOrganizationHeader({
          query: { domainType: "DOMAIN_TYPE_ORGANIZATION", domainId: organizationId },
        }),
      );
      return response.data?.roles || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useOrganizationGroups = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.groups(organizationId),
    queryFn: async () => {
      const response = await groupsListGroups(
        withOrganizationHeader({
          query: { domainId: organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
        }),
      );
      return response.data?.groups || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useOrganizationGroup = (organizationId: string, groupName: string) => {
  return useQuery({
    queryKey: organizationKeys.group(organizationId, groupName),
    queryFn: async () => {
      const response = await groupsDescribeGroup(
        withOrganizationHeader({
          path: { groupName },
          query: { domainId: organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
        }),
      );
      return response.data?.group || null;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!groupName,
  });
};

export const useOrganizationGroupUsers = (organizationId: string, groupName: string) => {
  return useQuery({
    queryKey: organizationKeys.groupUsers(organizationId, groupName),
    queryFn: async () => {
      const response = await groupsListGroupUsers(
        withOrganizationHeader({
          path: { groupName },
          query: { domainId: organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
        }),
      );
      return response.data?.users || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!groupName,
  });
};

export const useRole = (organizationId: string, roleName: string) => {
  return useQuery({
    queryKey: organizationKeys.role(organizationId, roleName),
    queryFn: async () => {
      const response = await rolesDescribeRole(
        withOrganizationHeader({
          path: {
            roleName: roleName,
          },
          query: {
            domainType: "DOMAIN_TYPE_ORGANIZATION",
            domainId: organizationId,
          },
        }),
      );
      return response.data?.role || null;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!roleName,
  });
};

export const useOrganizationInviteLink = (organizationId: string, enabled = true) => {
  return useQuery({
    queryKey: organizationKeys.inviteLink(organizationId),
    queryFn: async () => {
      const response = await organizationsGetInviteLink(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data?.inviteLink || null;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId && enabled,
  });
};

export const useOrganizationAgentSettings = (organizationId: string, enabled = true) => {
  return useQuery({
    queryKey: organizationKeys.agentSettings(organizationId),
    queryFn: async () => {
      const response = await organizationsGetAgentSettings(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data?.agentSettings || null;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId && enabled,
  });
};

export const useOrganizationUsage = (organizationId: string, enabled = true) => {
  return useQuery({
    queryKey: organizationKeys.usage(organizationId),
    queryFn: async () => {
      const response = await organizationsDescribeUsage(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data || null;
    },
    staleTime: 30 * 1000,
    gcTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
    enabled: !!organizationId && enabled,
  });
};
