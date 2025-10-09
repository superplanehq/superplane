import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  workflowsListWorkflows,
  workflowsDescribeWorkflow,
  workflowsCreateWorkflow,
  workflowsUpdateWorkflow,
  workflowsListNodeExecutions,
  workflowsListWorkflowEvents,
  workflowsListEventExecutions,
} from '../api-client/sdk.gen'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'

// Query Keys
export const workflowKeys = {
  all: ['workflows'] as const,
  lists: () => [...workflowKeys.all, 'list'] as const,
  list: (orgId: string) => [...workflowKeys.lists(), orgId] as const,
  details: () => [...workflowKeys.all, 'detail'] as const,
  detail: (orgId: string, id: string) => [...workflowKeys.details(), orgId, id] as const,
  nodeExecutions: () => [...workflowKeys.all, 'nodeExecutions'] as const,
  nodeExecution: (workflowId: string, nodeId: string, states?: string[]) =>
    [...workflowKeys.nodeExecutions(), workflowId, nodeId, ...(states || [])] as const,
  events: () => [...workflowKeys.all, 'events'] as const,
  eventList: (workflowId: string) => [...workflowKeys.events(), workflowId] as const,
  eventExecutions: () => [...workflowKeys.all, 'eventExecutions'] as const,
  eventExecution: (workflowId: string, eventId: string) =>
    [...workflowKeys.eventExecutions(), workflowId, eventId] as const,
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

export const useNodeExecutions = (
  workflowId: string,
  nodeId: string,
  options?: {
    states?: string[]
  }
) => {
  return useQuery({
    queryKey: workflowKeys.nodeExecution(workflowId, nodeId, options?.states),
    queryFn: async () => {
      const response = await workflowsListNodeExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
            nodeId,
          },
          query: options?.states ? {
            states: options.states,
          } : undefined,
        })
      )
      return response.data
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!workflowId && !!nodeId,
  })
}

export const useWorkflowEvents = (workflowId: string) => {
  return useQuery({
    queryKey: workflowKeys.eventList(workflowId),
    queryFn: async () => {
      const response = await workflowsListWorkflowEvents(
        withOrganizationHeader({
          path: { workflowId }
        })
      )
      return response.data
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!workflowId,
  })
}

export const useEventExecutions = (workflowId: string, eventId: string | null) => {
  return useQuery({
    queryKey: workflowKeys.eventExecution(workflowId, eventId!),
    queryFn: async () => {
      const response = await workflowsListEventExecutions(
        withOrganizationHeader({
          path: {
            workflowId,
            eventId: eventId!,
          }
        })
      )
      return response.data
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!workflowId && !!eventId,
  })
}
