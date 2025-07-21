import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  authorizationGetOrganizationUsers,
  authorizationListRoles,
  authorizationListOrganizationGroups,
  authorizationGetOrganizationGroup,
  authorizationGetOrganizationGroupUsers,
  authorizationAssignRole,
  authorizationRemoveRole,
  authorizationCreateOrganizationGroup,
  authorizationUpdateOrganizationGroup,
  authorizationDeleteOrganizationGroup,
  authorizationAddUserToOrganizationGroup,
  authorizationRemoveUserFromOrganizationGroup,
  authorizationCreateRole,
  authorizationUpdateRole,
  authorizationDeleteRole,
  authorizationDescribeRole,
  organizationsDescribeOrganization,
  organizationsUpdateOrganization,
  superplaneListCanvases,
  superplaneCreateCanvas
} from '../api-client/sdk.gen'
import { AuthorizationCreateRoleRequest, AuthorizationDomainType, AuthorizationRoleAssignment } from '@/api-client'

// Query Keys
export const organizationKeys = {
  all: ['organization'] as const,
  details: (orgId: string) => [...organizationKeys.all, 'details', orgId] as const,
  users: (orgId: string) => [...organizationKeys.all, 'users', orgId] as const,
  roles: (orgId: string) => [...organizationKeys.all, 'roles', orgId] as const,
  groups: (orgId: string) => [...organizationKeys.all, 'groups', orgId] as const,
  group: (orgId: string, groupName: string) => [...organizationKeys.all, 'group', orgId, groupName] as const,
  groupUsers: (orgId: string, groupName: string) => [...organizationKeys.all, 'groupUsers', orgId, groupName] as const,
  role: (orgId: string, roleName: string) => [...organizationKeys.all, 'role', orgId, roleName] as const,
  canvases: (orgId: string) => [...organizationKeys.all, 'canvases', orgId] as const,
}

// Hooks for fetching data
export const useOrganization = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.details(organizationId),
    queryFn: async () => {
      const response = await organizationsDescribeOrganization({
        path: { idOrName: organizationId }
      })
      return response.data?.organization || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId,
  })
}

export const useOrganizationUsers = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.users(organizationId),
    queryFn: async () => {
      const response = await authorizationGetOrganizationUsers({
        path: { organizationId }
      })
      return response.data?.users || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}

export const useOrganizationRoles = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.roles(organizationId),
    queryFn: async () => {
      const response = await authorizationListRoles({
        query: { domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: organizationId }
      })
      return response.data?.roles || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}

export const useOrganizationGroups = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.groups(organizationId),
    queryFn: async () => {
      const response = await authorizationListOrganizationGroups({
        query: { organizationId }
      })
      return response.data?.groups || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}

export const useOrganizationGroup = (organizationId: string, groupName: string) => {
  return useQuery({
    queryKey: organizationKeys.group(organizationId, groupName),
    queryFn: async () => {
      const response = await authorizationGetOrganizationGroup({
        path: { groupName },
        query: { organizationId }
      })
      return response.data?.group || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!groupName,
  })
}

export const useOrganizationGroupUsers = (organizationId: string, groupName: string) => {
  return useQuery({
    queryKey: organizationKeys.groupUsers(organizationId, groupName),
    queryFn: async () => {
      const response = await authorizationGetOrganizationGroupUsers({
        path: { groupName },
        query: { organizationId }
      })
      return response.data?.users || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!groupName,
  })
}

export const useRole = (organizationId: string, roleName: string) => {
  return useQuery({
    queryKey: organizationKeys.role(organizationId, roleName),
    queryFn: async () => {
      const response = await authorizationDescribeRole({
        query: {
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId,
          role: roleName
        }
      })
      return response.data?.role || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!roleName,
  })
}

// Mutations with cache invalidation
export const useAssignRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId?: string, 
      userEmail?: string, 
      roleAssignment: AuthorizationRoleAssignment 
    }) => {
      return await authorizationAssignRole({
        body: {
          userId: params.userId,
          userEmail: params.userEmail,
          roleAssignment: params.roleAssignment
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch organization users
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) })
    }
  })
}

export const useRemoveRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId: string, 
      roleAssignment: AuthorizationRoleAssignment
    }) => {
      return await authorizationRemoveRole({
        body: {
          userId: params.userId,
          roleAssignment: params.roleAssignment
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch organization users
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) })
    }
  })
}

export const useCreateGroup = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      organizationId: string, 
      groupName: string, 
      displayName?: string,
      description?: string,
      role?: string 
    }) => {
      return await authorizationCreateOrganizationGroup({
        body: {
          organizationId: params.organizationId,
          groupName: params.groupName,
          displayName: params.displayName,
          description: params.description,
          role: params.role
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch groups
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) })
    }
  })
}

export const useUpdateGroup = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      groupName: string, 
      organizationId: string, 
      displayName?: string,
      description?: string,
      role?: string 
    }) => {
      return await authorizationUpdateOrganizationGroup({
        path: { groupName: params.groupName },
        body: {
          organizationId: params.organizationId,
          displayName: params.displayName,
          description: params.description,
          role: params.role
        }
      })
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch groups and specific group data
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) })
      queryClient.invalidateQueries({ queryKey: organizationKeys.group(organizationId, variables.groupName) })
    }
  })
}

export const useDeleteGroup = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { groupName: string, organizationId: string }) => {
      return await authorizationDeleteOrganizationGroup({
        path: { groupName: params.groupName },
        query: { organizationId: params.organizationId }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch groups
      queryClient.invalidateQueries({ queryKey: organizationKeys.groups(organizationId) })
    }
  })
}

export const useAddUserToGroup = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      groupName: string, 
      userId?: string, 
      userEmail?: string, 
      organizationId: string 
    }) => {
      return await authorizationAddUserToOrganizationGroup({
        path: { groupName: params.groupName },
        body: {
          userId: params.userId,
          userEmail: params.userEmail,
          organizationId: params.organizationId
        }
      })
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch group users and organization users
      queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(organizationId, variables.groupName) })
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) })
    }
  })
}

export const useRemoveUserFromGroup = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      groupName: string, 
      userId: string, 
      organizationId: string 
    }) => {
      return await authorizationRemoveUserFromOrganizationGroup({
        path: { groupName: params.groupName },
        body: {
          userId: params.userId,
          organizationId: params.organizationId
        }
      })
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch group users
      queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(organizationId, variables.groupName) })
    }
  })
}

export const useCreateRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: AuthorizationCreateRoleRequest) => {
      return await authorizationCreateRole({
        body: params
      })
    },
    onSuccess: () => {
      // Invalidate and refetch roles
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) })
    }
  })
}

export const useUpdateRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      roleName: string,
      domainType: AuthorizationDomainType | undefined,
      domainId: string,
      permissions: Array<{ resource: string, action: string, domainType: AuthorizationDomainType | undefined }>,
      displayName?: string,
      description?: string
    }) => {
      return await authorizationUpdateRole({
        path: { roleName: params.roleName },
        body: {
          domainType: params.domainType,
          domainId: params.domainId,
          permissions: params.permissions,
          displayName: params.displayName,
          description: params.description
        }
      })
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch roles and specific role data
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) })
      queryClient.invalidateQueries({ queryKey: organizationKeys.role(organizationId, variables.roleName) })
    }
  })
}

export const useDeleteRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      roleName: string,
      domainType: AuthorizationDomainType,
      domainId: string
    }) => {
      return await authorizationDeleteRole({
        path: { roleName: params.roleName },
        query: {
          domainType: params.domainType,
          domainId: params.domainId
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch roles
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) })
    }
  })
}

export const useUpdateOrganization = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      displayName?: string,
      description?: string
    }) => {
      return await organizationsUpdateOrganization({
        path: { idOrName: organizationId },
        body: {
          organization: {
            metadata: {
              displayName: params.displayName,
              description: params.description
            }
          }
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch organization details
      queryClient.invalidateQueries({ queryKey: organizationKeys.details(organizationId) })
    }
  })
}

export const useOrganizationCanvases = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.canvases(organizationId),
    queryFn: async () => {
      const response = await superplaneListCanvases({ query: { organizationId } })
      return response.data?.canvases || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId,
  })
}

export const useCreateCanvas = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      canvas: {
        metadata: {
          name: string
          description?: string
        }
      },
      organizationId: string
    }) => {
      return await superplaneCreateCanvas({
        body: params
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvases
      queryClient.invalidateQueries({ queryKey: organizationKeys.canvases(organizationId) })
    }
  })
}