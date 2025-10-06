import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  blueprintsListBlueprints,
  blueprintsDescribeBlueprint,
  blueprintsCreateBlueprint,
  blueprintsUpdateBlueprint,
  primitivesListPrimitives,
  primitivesDescribePrimitive,
} from '../api-client/sdk.gen'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'

// Query Keys
export const blueprintKeys = {
  all: ['blueprints'] as const,
  lists: () => [...blueprintKeys.all, 'list'] as const,
  list: (orgId: string) => [...blueprintKeys.lists(), orgId] as const,
  details: () => [...blueprintKeys.all, 'detail'] as const,
  detail: (orgId: string, id: string) => [...blueprintKeys.details(), orgId, id] as const,
}

export const primitiveKeys = {
  all: ['primitives'] as const,
  lists: () => [...primitiveKeys.all, 'list'] as const,
  list: (orgId: string) => [...primitiveKeys.lists(), orgId] as const,
  details: () => [...primitiveKeys.all, 'detail'] as const,
  detail: (orgId: string, name: string) => [...primitiveKeys.details(), orgId, name] as const,
}

// Hooks for fetching blueprints
export const useBlueprints = (organizationId: string) => {
  return useQuery({
    queryKey: blueprintKeys.list(organizationId),
    queryFn: async () => {
      const response = await blueprintsListBlueprints(
        withOrganizationHeader({})
      )
      return response.data?.blueprints || []
    },
    enabled: !!organizationId,
  })
}

export const useBlueprint = (organizationId: string, blueprintId: string) => {
  return useQuery({
    queryKey: blueprintKeys.detail(organizationId, blueprintId),
    queryFn: async () => {
      const response = await blueprintsDescribeBlueprint(
        withOrganizationHeader({
          path: { id: blueprintId }
        })
      )
      return response.data?.blueprint
    },
    enabled: !!organizationId && !!blueprintId,
  })
}

export const useCreateBlueprint = (organizationId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      const payload = {
        name: data.name,
        description: data.description || '',
        nodes: data.nodes || [],
        edges: data.edges || [],
      }

      return await blueprintsCreateBlueprint(
        withOrganizationHeader({
          body: {
            blueprint: payload
          }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: blueprintKeys.list(organizationId) })
    },
  })
}

export const useUpdateBlueprint = (organizationId: string, blueprintId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      return await blueprintsUpdateBlueprint(
        withOrganizationHeader({
          path: { id: blueprintId },
          body: {
            blueprint: {
              name: data.name,
              description: data.description || '',
              nodes: data.nodes || [],
              edges: data.edges || [],
            }
          }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: blueprintKeys.detail(organizationId, blueprintId) })
      queryClient.invalidateQueries({ queryKey: blueprintKeys.list(organizationId) })
    },
  })
}

// Hooks for fetching primitives
export const usePrimitives = (organizationId: string) => {
  return useQuery({
    queryKey: primitiveKeys.list(organizationId),
    queryFn: async () => {
      const response = await primitivesListPrimitives(
        withOrganizationHeader({})
      )
      return response.data?.primitives || []
    },
    enabled: !!organizationId,
  })
}

export const usePrimitive = (organizationId: string, primitiveName: string) => {
  return useQuery({
    queryKey: primitiveKeys.detail(organizationId, primitiveName),
    queryFn: async () => {
      const response = await primitivesDescribePrimitive(
        withOrganizationHeader({
          path: { name: primitiveName }
        })
      )
      return response.data?.primitive
    },
    enabled: !!organizationId && !!primitiveName,
  })
}
