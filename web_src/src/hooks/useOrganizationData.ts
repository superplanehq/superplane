import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  usersListUsers,
  rolesListRoles,
  groupsListGroups,
  groupsDescribeGroup,
  groupsListGroupUsers,
  rolesAssignRole,
  organizationsRemoveUser,
  groupsCreateGroup,
  groupsUpdateGroup,
  groupsDeleteGroup,
  groupsAddUserToGroup,
  groupsRemoveUserFromGroup,
  rolesCreateRole,
  rolesUpdateRole,
  rolesDeleteRole,
  rolesDescribeRole,
  organizationsDescribeOrganization,
  organizationsUpdateOrganization,
  superplaneListCanvases,
  superplaneCreateCanvas,
  superplaneDeleteCanvas
} from '../api-client/sdk.gen'
import { RolesCreateRoleRequest, AuthorizationDomainType } from '@/api-client'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'

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
      const response = await organizationsDescribeOrganization(
        withOrganizationHeader({
          path: { id: organizationId }
        })
      )
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
      const response = await usersListUsers(
        withOrganizationHeader({
          query: { domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: organizationId }
        })
      )
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
      const response = await rolesListRoles(
        withOrganizationHeader({
          query: { domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: organizationId }
        })
      )
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
      const response = await groupsListGroups(
        withOrganizationHeader({
          query: { domainId: organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' }
        })
      )
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
      const response = await groupsDescribeGroup(
        withOrganizationHeader({
          path: { groupName },
          query: { domainId: organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' }
        })
      )
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
      const response = await groupsListGroupUsers(
        withOrganizationHeader({
          path: { groupName },
          query: { domainId: organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' }
        })
      )
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
      const response = await rolesDescribeRole(
        withOrganizationHeader({
          query: {
            domainType: 'DOMAIN_TYPE_ORGANIZATION',
            domainId: organizationId,
            role: roleName
          }
        })
      )
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
      roleName: string,
    }) => {
      return await rolesAssignRole(
        withOrganizationHeader({
          path: {
            roleName: params.roleName,
          },
          body: {
            userId: params.userId,
            userEmail: params.userEmail,
            domainType: 'DOMAIN_TYPE_ORGANIZATION',
            domainId: organizationId
          },
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) })
    }
  })
}

export const useRemoveOrganizationUser = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId: string,
    }) => {
      return await organizationsRemoveUser(
        withOrganizationHeader({
          path: {
            id: organizationId,
            userId: params.userId
          }
        })
      )
    },
    onSuccess: () => {
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
                role: params.role
              }
            },
            domainId: params.organizationId,
            domainType: 'DOMAIN_TYPE_ORGANIZATION',
          }
        })
      )
    },
    onSuccess: () => {
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
                role: params.role
              }
            },
            domainId: params.organizationId,
            domainType: 'DOMAIN_TYPE_ORGANIZATION',
          }
        })
      )
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
      return await groupsDeleteGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          query: { domainId: params.organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' }
        })
      )
    },
    onSuccess: () => {
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
      return await groupsAddUserToGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          body: {
            userId: params.userId,
            userEmail: params.userEmail,
            domainId: params.organizationId,
            domainType: 'DOMAIN_TYPE_ORGANIZATION'
          }
        })
      )
    },
    onSuccess: (_, variables) => {
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
      return await groupsRemoveUserFromGroup(
        withOrganizationHeader({
          path: { groupName: params.groupName },
          body: {
            userId: params.userId,
            domainId: params.organizationId,
            domainType: 'DOMAIN_TYPE_ORGANIZATION'
          }
        })
      )
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(organizationId, variables.groupName) })
    }
  })
}

export const useCreateRole = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: RolesCreateRoleRequest) => {
      return await rolesCreateRole(
        withOrganizationHeader({
          body: params
        })
      )
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
                description: params.description
              }
            }
          }
        })
      )
    },
    onSuccess: (_, variables) => {
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
      return await rolesDeleteRole(
        withOrganizationHeader({
          path: { roleName: params.roleName },
          query: {
            domainType: params.domainType,
            domainId: params.domainId
          }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.roles(organizationId) })
    }
  })
}

export const useUpdateOrganization = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      name?: string,
      description?: string
    }) => {
      return await organizationsUpdateOrganization(
        withOrganizationHeader({
          path: { id: organizationId },
          body: {
            organization: {
              metadata: {
                name: params.name,
                description: params.description
              }
            }
          }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.details(organizationId) })
    }
  })
}

export const useOrganizationCanvases = (organizationId: string) => {
  return useQuery({
    queryKey: organizationKeys.canvases(organizationId),
    queryFn: async () => {
      const response = await superplaneListCanvases(
        withOrganizationHeader({ query: { organizationId } })
      )
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
      return await superplaneCreateCanvas(
        withOrganizationHeader({
          body: params
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.canvases(organizationId) })
    }
  })
}

export const useDeleteCanvas = (organizationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      canvasId: string
    }) => {
      return await superplaneDeleteCanvas(
        withOrganizationHeader({
          path: { idOrName: params.canvasId },
          query: { organizationId }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.canvases(organizationId) })
    }
  })
}
  
