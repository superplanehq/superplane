import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  blueprintsListBlueprints,
  blueprintsDescribeBlueprint,
  blueprintsCreateBlueprint,
  blueprintsUpdateBlueprint,
  blueprintsDeleteBlueprint,
  componentsListComponents,
  componentsDescribeComponent,
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

export const componentKeys = {
  all: ['components'] as const,
  lists: () => [...componentKeys.all, 'list'] as const,
  list: (orgId: string) => [...componentKeys.lists(), orgId] as const,
  details: () => [...componentKeys.all, 'detail'] as const,
  detail: (orgId: string, name: string) => [...componentKeys.details(), orgId, name] as const,
}

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
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[]; configuration?: any[]; outputChannels?: any[]; icon?: string; color?: string }) => {
      const payload = {
        name: data.name,
        description: data.description || '',
        nodes: data.nodes || [],
        edges: data.edges || [],
        configuration: data.configuration || [],
        outputChannels: data.outputChannels || [],
        icon: data.icon,
        color: data.color,
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
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[]; configuration?: any[]; outputChannels?: any[]; icon?: string; color?: string }) => {
      return await blueprintsUpdateBlueprint(
        withOrganizationHeader({
          path: { id: blueprintId },
          body: {
            blueprint: {
              name: data.name,
              description: data.description || '',
              nodes: data.nodes || [],
              edges: data.edges || [],
              configuration: data.configuration || [],
              outputChannels: data.outputChannels || [],
              icon: data.icon,
              color: data.color,
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

export const useDeleteBlueprint = (organizationId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (blueprintId: string) => {
      return await blueprintsDeleteBlueprint(
        withOrganizationHeader({
          path: { id: blueprintId }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: blueprintKeys.list(organizationId) })
    },
  })
}

export const useComponents = (organizationId: string) => {
  return useQuery({
    queryKey: componentKeys.list(organizationId),
    queryFn: async () => {
      const response = await componentsListComponents(
        withOrganizationHeader({})
      )
      return response.data?.components || []
    },
    enabled: !!organizationId,
  })
}

export const useComponent = (organizationId: string, componentName: string) => {
  return useQuery({
    queryKey: componentKeys.detail(organizationId, componentName),
    queryFn: async () => {
      const response = await componentsDescribeComponent(
        withOrganizationHeader({
          path: { name: componentName }
        })
      )
      return response.data?.component
    },
    enabled: !!organizationId && !!componentName,
  })
}
