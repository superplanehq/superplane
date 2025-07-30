import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
  rolesRemoveRole,
  superplaneListStages,
  superplaneCreateStage,
  superplaneUpdateStage,
  superplaneDescribeStage,
  superplaneCreateEventSource,
  superplaneDescribeEventSource,
  integrationsListIntegrations,
} from '../api-client/sdk.gen'
import type { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneCondition, IntegrationsResourceRef, SuperplaneEventSourceSpec, SuperplaneValueDefinition } from '../api-client/types.gen'

export const canvasKeys = {
  all: ['canvas'] as const,
  details: (canvasId: string) => [...canvasKeys.all, 'details', canvasId] as const,
  users: (canvasId: string) => [...canvasKeys.all, 'users', canvasId] as const,
  roles: (canvasId: string) => [...canvasKeys.all, 'roles', canvasId] as const,
  stages: (canvasId: string) => [...canvasKeys.all, 'stages', canvasId] as const,
  stage: (canvasId: string, stageId: string) => [...canvasKeys.all, 'stage', canvasId, stageId] as const,
  eventSources: (canvasId: string) => [...canvasKeys.all, 'eventSources', canvasId] as const,
  eventSource: (canvasId: string, eventSourceId: string) => [...canvasKeys.all, 'eventSource', canvasId, eventSourceId] as const,
  integrations: (canvasId?: string) => canvasId ? [...canvasKeys.all, 'integrations', canvasId] as const : ['integrations'] as const,
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

// Stage-related hooks
export const useCanvasStages = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.stages(canvasId),
    queryFn: async () => {
      const response = await superplaneListStages({
        path: { canvasIdOrName: canvasId }
      })
      return response.data?.stages || []
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId
  })
}

export const useStageDetails = (canvasId: string, stageId: string) => {
  return useQuery({
    queryKey: canvasKeys.stage(canvasId, stageId),
    queryFn: async () => {
      const response = await superplaneDescribeStage({
        path: { canvasIdOrName: canvasId, id: stageId }
      })
      return response.data?.stage
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!stageId
  })
}

export const useCreateStage = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (stageData: {
      name: string;
      description?: string;
      inputs?: SuperplaneInputDefinition[];
      outputs?: SuperplaneOutputDefinition[];
      connections?: SuperplaneConnection[];
      executor?: SuperplaneExecutor;
      secrets?: SuperplaneValueDefinition[];
      conditions?: SuperplaneCondition[];
    }) => {
      return await superplaneCreateStage({
        path: { canvasIdOrName: canvasId },
        body: {
          stage: {
            metadata: {
              name: stageData.name,
              canvasId: canvasId,
              // description: stageData.description,
            },
            spec: {
              inputs: stageData.inputs || [],
              outputs: stageData.outputs || [],
              connections: stageData.connections || [],
              executor: stageData.executor,
              secrets: stageData.secrets || [],
              conditions: stageData.conditions || []
            }
          }
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvas stages
      queryClient.invalidateQueries({ queryKey: canvasKeys.stages(canvasId) })
    }
  })
}

// Helper hook for creating stages with integrations
export const useCreateStageWithIntegration = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (stageData: {
      name: string;
      inputs?: SuperplaneInputDefinition[];
      outputs?: SuperplaneOutputDefinition[];
      connections?: SuperplaneConnection[];
      executor?: {
        type: string;
        integration?: {
          name: string;
        };
        resource?: IntegrationsResourceRef;
        spec?: { [key: string]: unknown };
      };
      conditions?: SuperplaneCondition[];
    }) => {
      return await superplaneCreateStage({
        path: { canvasIdOrName: canvasId },
        body: {
          stage: {
            metadata: {
              name: stageData.name,
              canvasId: canvasId
            },
            spec: {
              inputs: stageData.inputs || [],
              outputs: stageData.outputs || [],
              connections: stageData.connections || [],
              executor: stageData.executor,
              conditions: stageData.conditions || []
            }
          }
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvas stages
      queryClient.invalidateQueries({ queryKey: canvasKeys.stages(canvasId) })
    }
  })
}

export const useUpdateStage = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      stageId: string;
      name: string;
      description?: string;
      inputs?: SuperplaneInputDefinition[];
      outputs?: SuperplaneOutputDefinition[];
      connections?: SuperplaneConnection[];
      executor?: SuperplaneExecutor;
      secrets?: SuperplaneValueDefinition[];
      conditions?: SuperplaneCondition[];
    }) => {
      return await superplaneUpdateStage({
        path: { canvasIdOrName: canvasId, idOrName: params.stageId },
        body: {
          stage: {
            metadata: {
              name: params.name,
              canvasId: canvasId,
              // description: params.description,
            },
            spec: {
              inputs: params.inputs || [],
              outputs: params.outputs || [],
              connections: params.connections || [],
              executor: params.executor,
              secrets: params.secrets || [],
              conditions: params.conditions || []
            }
          }
        }
      })
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch canvas stages and specific stage
      queryClient.invalidateQueries({ queryKey: canvasKeys.stages(canvasId) })
      queryClient.invalidateQueries({ queryKey: canvasKeys.stage(canvasId, variables.stageId) })
    }
  })
}

// Event Source-related hooks
export const useEventSourceDetails = (canvasId: string, eventSourceId: string) => {
  return useQuery({
    queryKey: canvasKeys.eventSource(canvasId, eventSourceId),
    queryFn: async () => {
      const response = await superplaneDescribeEventSource({
        path: { canvasIdOrName: canvasId, id: eventSourceId }
      })
      return response.data?.eventSource
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!eventSourceId
  })
}

export const useCreateEventSource = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (eventSourceData: {
      name: string;
      spec: SuperplaneEventSourceSpec;
    }) => {
      return await superplaneCreateEventSource({
        path: { canvasIdOrName: canvasId },
        body: {
          eventSource: {
            metadata: {
              name: eventSourceData.name,
              canvasId: canvasId
            },
            spec: eventSourceData.spec
          }
        }
      })
    },
    onSuccess: () => {
      // Invalidate and refetch canvas event sources
      queryClient.invalidateQueries({ queryKey: canvasKeys.eventSources(canvasId) })
    }
  })
}

// Integration-related hooks
export const useIntegrations = () => {
  return useQuery({
    queryKey: canvasKeys.integrations(),
    queryFn: async () => {
      const response = await integrationsListIntegrations()
      return response.data?.integrations || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}