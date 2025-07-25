import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
  rolesRemoveRole,
} from '../api-client/sdk.gen'

export const canvasKeys = {
  all: ['canvas'] as const,
  details: (canvasId: string) => [...canvasKeys.all, 'details', canvasId] as const,
  users: (canvasId: string) => [...canvasKeys.all, 'users', canvasId] as const,
  roles: (canvasId: string) => [...canvasKeys.all, 'roles', canvasId] as const,
}

export const useCanvasRoles = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.roles(canvasId),
    queryFn: async () => {
      const response = await rolesListRoles({
        query: { domainId: canvasId, domainType: 'DOMAIN_TYPE_CANVAS' },
      })
      return response.data?.roles || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId
  })
}

export const useCanvasUsers = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.users(canvasId),
    queryFn: async () => {
      const response = await usersListUsers({
        query: { domainId: canvasId, domainType: 'DOMAIN_TYPE_CANVAS' },
      })
      return response.data?.users || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId
  })
}

export const useOrganizationUsersForCanvas = (organizationId: string) => {
  return useQuery({
    queryKey: ['organizationUsers', organizationId],
    queryFn: async () => {
      const response = await usersListUsers({
        query: { domainId: organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' },
      })
      return response.data?.users || []
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!organizationId
  })
}

export const useAssignCanvasRole = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId?: string, 
      userEmail?: string,
      role: string,
    }) => {
      return await rolesAssignRole({
        body: {
          userId: params.userId,
          userEmail: params.userEmail,
          roleName: params.role,
          domainId: canvasId,
          domainType: 'DOMAIN_TYPE_CANVAS'
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) })
    }
  })
}

export const useRemoveCanvasRole = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId: string, 
      role: string,
    }) => {
      return await rolesRemoveRole({
        body: {
          userId: params.userId,
          roleName: params.role,
          domainId: canvasId,
          domainType: 'DOMAIN_TYPE_CANVAS'
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) })
    }
  })
}