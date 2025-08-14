import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
  superplaneListStages,
  superplaneCreateStage,
  superplaneUpdateStage,
  superplaneDescribeStage,
  superplaneCreateEventSource,
  superplaneDescribeEventSource,
  superplaneListConnectionGroups,
  superplaneCreateConnectionGroup,
  superplaneUpdateConnectionGroup,
  superplaneDescribeConnectionGroup,
  integrationsListIntegrations,
  superplaneAddUser,
  superplaneRemoveUser,
} from '../api-client/sdk.gen'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'
import type { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneCondition, IntegrationsResourceRef, SuperplaneEventSourceSpec, SuperplaneValueDefinition, GroupByField, SpecTimeoutBehavior, SuperplaneInputMapping } from '../api-client/types.gen'

export const canvasKeys = {
  all: ['canvas'] as const,
  details: (canvasId: string) => [...canvasKeys.all, 'details', canvasId] as const,
  users: (canvasId: string) => [...canvasKeys.all, 'users', canvasId] as const,
  roles: (canvasId: string) => [...canvasKeys.all, 'roles', canvasId] as const,
  stages: (canvasId: string) => [...canvasKeys.all, 'stages', canvasId] as const,
  stage: (canvasId: string, stageId: string) => [...canvasKeys.all, 'stage', canvasId, stageId] as const,
  eventSources: (canvasId: string) => [...canvasKeys.all, 'eventSources', canvasId] as const,
  eventSource: (canvasId: string, eventSourceId: string) => [...canvasKeys.all, 'eventSource', canvasId, eventSourceId] as const,
  connectionGroups: (canvasId: string) => [...canvasKeys.all, 'connectionGroups', canvasId] as const,
  connectionGroup: (canvasId: string, connectionGroupId: string) => [...canvasKeys.all, 'connectionGroup', canvasId, connectionGroupId] as const,
  integrations: (canvasId?: string) => canvasId ? [...canvasKeys.all, 'integrations', canvasId] as const : ['integrations'] as const,
}

export const useCanvasRoles = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.roles(canvasId),
    queryFn: async () => {
      const response = await rolesListRoles(withOrganizationHeader({
        query: { domainId: canvasId, domainType: 'DOMAIN_TYPE_CANVAS' },
      }))
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
      const response = await usersListUsers(withOrganizationHeader({
        query: { domainId: canvasId, domainType: 'DOMAIN_TYPE_CANVAS' },
      }))
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
      const response = await usersListUsers(withOrganizationHeader({
        query: { domainId: organizationId, domainType: 'DOMAIN_TYPE_ORGANIZATION' },
      }))
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
      return await rolesAssignRole(withOrganizationHeader({
        path: { roleName: params.role },
        body: {
          userId: params.userId,
          userEmail: params.userEmail,
          domainId: canvasId,
          domainType: 'DOMAIN_TYPE_CANVAS'
        }
      }))
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) })
    }
  })
}

export const useAddCanvasUser = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId: string,
    }) => {
      return await superplaneAddUser(withOrganizationHeader({
        path: { canvasIdOrName: canvasId },
        body: { userId: params.userId }
      }))
    },
    onSuccess: () => {
      // Invalidate and refetch canvas users
      queryClient.invalidateQueries({ queryKey: canvasKeys.users(canvasId) })
    }
  })
}

export const useRemoveCanvasUser = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      userId: string,
    }) => {
      return await superplaneRemoveUser(withOrganizationHeader({
        path: { canvasIdOrName: canvasId, userId: params.userId }
      }))
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
      const response = await superplaneListStages(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId }
        })
      )
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
      const response = await superplaneDescribeStage(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: stageId }
        })
      )
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
      inputMappings?: SuperplaneInputMapping[];
    }) => {
      return await superplaneCreateStage(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          body: {
            stage: {
              metadata: {
                name: stageData.name,
                canvasId: canvasId,
                description: stageData.description,
              },
              spec: {
                inputs: stageData.inputs || [],
                outputs: stageData.outputs || [],
                connections: stageData.connections || [],
                executor: stageData.executor,
                secrets: stageData.secrets || [],
                conditions: stageData.conditions || [],
                inputMappings: stageData.inputMappings || []
              }
            }
          }
        })
      )
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
      return await superplaneCreateStage(
        withOrganizationHeader({
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
      )
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
      inputMappings?: SuperplaneInputMapping[];
    }) => {
      return await superplaneUpdateStage(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: params.stageId },
          body: {
            stage: {
              metadata: {
                name: params.name,
                canvasId: canvasId,
                description: params.description,
              },
              spec: {
                inputs: params.inputs || [],
                outputs: params.outputs || [],
                connections: params.connections || [],
                executor: params.executor,
                secrets: params.secrets || [],
                conditions: params.conditions || [],
                inputMappings: params.inputMappings || []
              }
            }
          }
        })
      )
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
      const response = await superplaneDescribeEventSource(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: eventSourceId }
        })
      )
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
      description?: string;
      spec: SuperplaneEventSourceSpec;
    }) => {
      return await superplaneCreateEventSource(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          body: {
            eventSource: {
              metadata: {
                name: eventSourceData.name,
                description: eventSourceData.description,
                canvasId: canvasId
              },
              spec: eventSourceData.spec
            }
          }
        })
      )
    },
    onSuccess: () => {
      // Invalidate and refetch canvas event sources
      queryClient.invalidateQueries({ queryKey: canvasKeys.eventSources(canvasId) })
    }
  })
}

// Connection Group-related hooks
export const useCanvasConnectionGroups = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.connectionGroups(canvasId),
    queryFn: async () => {
      const response = await superplaneListConnectionGroups(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId }
        })
      )
      return response.data?.connectionGroups || []
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId
  })
}

export const useConnectionGroupDetails = (canvasId: string, connectionGroupId: string) => {
  return useQuery({
    queryKey: canvasKeys.connectionGroup(canvasId, connectionGroupId),
    queryFn: async () => {
      const response = await superplaneDescribeConnectionGroup(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: connectionGroupId }
        })
      )
      return response.data?.connectionGroup
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!connectionGroupId
  })
}

export const useCreateConnectionGroup = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (connectionGroupData: {
      name: string;
      description?: string;
      connections: SuperplaneConnection[];
      groupByFields: GroupByField[];
      timeout?: number;
      timeoutBehavior?: SpecTimeoutBehavior;
    }) => {
      return await superplaneCreateConnectionGroup(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          body: {
            connectionGroup: {
              metadata: {
                name: connectionGroupData.name,
                description: connectionGroupData.description,
                canvasId: canvasId
              },
              spec: {
                connections: connectionGroupData.connections,
                groupBy: {
                  fields: connectionGroupData.groupByFields
                },
                timeout: connectionGroupData.timeout,
                timeoutBehavior: connectionGroupData.timeoutBehavior
              }
            }
          }
        })
      )
    },
    onSuccess: () => {
      // Invalidate and refetch canvas connection groups
      queryClient.invalidateQueries({ queryKey: canvasKeys.connectionGroups(canvasId) })
    }
  })
}

export const useUpdateConnectionGroup = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: {
      connectionGroupId: string;
      name: string;
      description?: string;
      connections: SuperplaneConnection[];
      groupByFields: GroupByField[];
      timeout?: number;
      timeoutBehavior?: SpecTimeoutBehavior;
    }) => {
      return await superplaneUpdateConnectionGroup(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: params.connectionGroupId },
          body: {
            connectionGroup: {
              metadata: {
                name: params.name,
                description: params.description,
                canvasId: canvasId
              },
              spec: {
                connections: params.connections,
                groupBy: {
                  fields: params.groupByFields
                },
                timeout: params.timeout,
                timeoutBehavior: params.timeoutBehavior
              }
            }
          }
        })
      )
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch canvas connection groups and specific connection group
      queryClient.invalidateQueries({ queryKey: canvasKeys.connectionGroups(canvasId) })
      queryClient.invalidateQueries({ queryKey: canvasKeys.connectionGroup(canvasId, variables.connectionGroupId) })
    }
  })
}

// Integration-related hooks
export const useIntegrations = () => {
  return useQuery({
    queryKey: canvasKeys.integrations(),
    queryFn: async () => {
      const response = await integrationsListIntegrations(withOrganizationHeader())
      return response.data?.integrations || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  })
}