import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  workflowsListWorkflows,
  workflowsDescribeWorkflow,
  workflowsCreateWorkflow,
  workflowsUpdateWorkflow,
} from '../api-client/sdk.gen'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'

// Query Keys
export const workflowKeys = {
  all: ['workflows'] as const,
  lists: () => [...workflowKeys.all, 'list'] as const,
  list: (orgId: string) => [...workflowKeys.lists(), orgId] as const,
  details: () => [...workflowKeys.all, 'detail'] as const,
  detail: (orgId: string, id: string) => [...workflowKeys.details(), orgId, id] as const,
}

// Hooks for fetching workflows
export const useWorkflows = (organizationId: string) => {
  return useQuery({
    queryKey: workflowKeys.list(organizationId),
    queryFn: async () => {
      const response = await workflowsListWorkflows(
        withOrganizationHeader({})
      )
      return response.data?.workflows || []
    },
    enabled: !!organizationId,
  })
}

export const useWorkflow = (organizationId: string, workflowId: string) => {
  return useQuery({
    queryKey: workflowKeys.detail(organizationId, workflowId),
    queryFn: async () => {
      const response = await workflowsDescribeWorkflow(
        withOrganizationHeader({
          path: { id: workflowId }
        })
      )
      return response.data?.workflow
    },
    enabled: !!organizationId && !!workflowId,
  })
}

export const useCreateWorkflow = (organizationId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      const payload = {
        name: data.name,
        description: data.description || '',
        nodes: data.nodes || [],
        edges: data.edges || [],
      }

      return await workflowsCreateWorkflow(
        withOrganizationHeader({
          body: {
            workflow: payload
          }
        })
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: workflowKeys.list(organizationId) })
    },
  })
}

export const useUpdateWorkflow = (organizationId: string, workflowId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: { name: string; description?: string; nodes?: any[]; edges?: any[] }) => {
      return await workflowsUpdateWorkflow(
        withOrganizationHeader({
          path: { id: workflowId },
          body: {
            workflow: {
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
      queryClient.invalidateQueries({ queryKey: workflowKeys.detail(organizationId, workflowId) })
      queryClient.invalidateQueries({ queryKey: workflowKeys.list(organizationId) })
    },
  })
}
