import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  authorizationListRoles,
  authorizationGetCanvasUsers,
  authorizationAssignRole,
  authorizationRemoveRole,
  authorizationGetOrganizationUsers
} from '../api-client/sdk.gen'
import type { AuthorizationRoleAssignment } from '../api-client/types.gen'

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
      const response = await authorizationListRoles({
        query: { domainType: 'DOMAIN_TYPE_CANVAS', domainId: canvasId }
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
      const response = await authorizationGetCanvasUsers({
        path: { canvasIdOrName: canvasId },
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
      const response = await authorizationGetOrganizationUsers({
        path: { organizationId }
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
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) })
    }
  })
}