import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query'
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
  rolesCreateRole,
  rolesUpdateRole,
  rolesDeleteRole,
  rolesDescribeRole,
  superplaneListStages,
  superplaneCreateStage,
  superplaneUpdateStage,
  superplaneDeleteStage,
  superplaneDescribeStage,
  superplaneCreateEventSource,
  superplaneUpdateEventSource,
  superplaneDeleteEventSource,
  superplaneDescribeEventSource,
  superplaneResetEventSourceKey,
  superplaneListConnectionGroups,
  superplaneCreateConnectionGroup,
  superplaneUpdateConnectionGroup,
  superplaneDeleteConnectionGroup,
  superplaneDescribeConnectionGroup,
  integrationsListIntegrations,
  superplaneAddUser,
  superplaneRemoveUser,
  superplaneListEvents,
  superplaneListStageEvents,
  superplaneListStageExecutions,
  superplaneListEventRejections,
  superplaneListAlerts,
  superplaneAcknowledgeAlert,
} from '../api-client/sdk.gen'
import { RolesCreateRoleRequest, AuthorizationDomainType } from '@/api-client'
import { withOrganizationHeader } from '../utils/withOrganizationHeader'
import type { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneCondition, IntegrationsResourceRef, SuperplaneEventSourceSpec, SuperplaneValueDefinition, GroupByField, SpecTimeoutBehavior, SuperplaneInputMapping, SuperplaneStageEventState, SuperplaneAlert } from '../api-client/types.gen'

export const canvasKeys = {
  all: ['canvas'] as const,
  alerts: (canvasId: string) => [...canvasKeys.all, 'alerts', canvasId] as const,
  details: (canvasId: string) => [...canvasKeys.all, 'details', canvasId] as const,
  users: (canvasId: string) => [...canvasKeys.all, 'users', canvasId] as const,
  roles: (canvasId: string) => [...canvasKeys.all, 'roles', canvasId] as const,
  stages: (canvasId: string) => [...canvasKeys.all, 'stages', canvasId] as const,
  stage: (canvasId: string, stageId: string) => [...canvasKeys.all, 'stage', canvasId, stageId] as const,
  eventSources: (canvasId: string) => [...canvasKeys.all, 'eventSources', canvasId] as const,
  eventSource: (canvasId: string, eventSourceId: string) => [...canvasKeys.all, 'eventSource', canvasId, eventSourceId] as const,
  events: (canvasId: string, sourceType: string, sourceId: string, states?: string[]) => [...canvasKeys.all, 'events', canvasId, sourceType, sourceId, states] as const,
  stageEvents: (canvasId: string, stageId: string, states: SuperplaneStageEventState[]) => [...canvasKeys.all, 'stageEvents', canvasId, stageId, states] as const,
  stageExecutions: (canvasId: string, stageId: string, results?: string[]) => [...canvasKeys.all, 'stageExecutions', canvasId, stageId, results] as const,
  connectionGroups: (canvasId: string) => [...canvasKeys.all, 'connectionGroups', canvasId] as const,
  connectionGroup: (canvasId: string, connectionGroupId: string) => [...canvasKeys.all, 'connectionGroup', canvasId, connectionGroupId] as const,
  integrations: (canvasId?: string) => canvasId ? [...canvasKeys.all, 'integrations', canvasId] as const : ['integrations'] as const,
  eventRejections: (canvasId: string, targetType: string, targetId: string) => [...canvasKeys.all, 'eventRejections', canvasId, targetType, targetId] as const,
  role: (canvasId: string, roleName: string) => [...canvasKeys.all, 'role', canvasId, roleName] as const,
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
      dryRun?: boolean;
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
                inputMappings: stageData.inputMappings || [],
                dryRun: stageData.dryRun || false
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
      dryRun?: boolean;
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
                inputMappings: params.inputMappings || [],
                dryRun: params.dryRun || false
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

export const useDeleteStage = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (stageId: string) => {
      return await superplaneDeleteStage(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: stageId }
        })
      )
    },
    onSuccess: (_, stageId) => {
      // Invalidate and refetch canvas stages
      queryClient.invalidateQueries({ queryKey: canvasKeys.stages(canvasId) })
      // Remove specific stage from cache
      queryClient.removeQueries({ queryKey: canvasKeys.stage(canvasId, stageId) })
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

export const useResetEventSourceKey = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (eventSourceId: string) => {
      return await superplaneResetEventSourceKey(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: eventSourceId },
        })
      )
    },
    onSuccess: () => {
      // Invalidate and refetch canvas event sources
      queryClient.invalidateQueries({ queryKey: canvasKeys.eventSources(canvasId) })
    }
  })
}

export const useUpdateEventSource = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: { 
      eventSourceId: string;
      name: string;
      description?: string;
      spec: SuperplaneEventSourceSpec;
    }) => {
      return await superplaneUpdateEventSource(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: params.eventSourceId },
          body: {
            eventSource: {
              metadata: {
                name: params.name,
                description: params.description
              },
              spec: params.spec
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

export const useDeleteEventSource = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (eventSourceId: string) => {
      return await superplaneDeleteEventSource(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: eventSourceId }
        })
      )
    },
    onSuccess: (_, eventSourceId) => {
      // Invalidate and refetch canvas event sources
      queryClient.invalidateQueries({ queryKey: canvasKeys.eventSources(canvasId) })
      // Remove specific event source from cache
      queryClient.removeQueries({ queryKey: canvasKeys.eventSource(canvasId, eventSourceId) })
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

export const useDeleteConnectionGroup = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (connectionGroupId: string) => {
      return await superplaneDeleteConnectionGroup(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, idOrName: connectionGroupId }
        })
      )
    },
    onSuccess: (_, connectionGroupId) => {
      // Invalidate and refetch canvas connection groups
      queryClient.invalidateQueries({ queryKey: canvasKeys.connectionGroups(canvasId) })
      // Remove specific connection group from cache
      queryClient.removeQueries({ queryKey: canvasKeys.connectionGroup(canvasId, connectionGroupId) })
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

// Event-related hooks
export const useEvents = (canvasId: string, sourceType: string, sourceId: string, states?: string[]) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.events(canvasId, sourceType, sourceId, states),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          query: {
            sourceType,
            sourceId,
            limit: 20,
            ...(pageParam && { before: pageParam }),
            ...(states && states.length > 0 && { states })
          }
        })
      )
      return {
        events: response.data?.events || [],
        totalCount: response.data?.totalCount || 0,
        hasNextPage: response.data?.hasNextPage || false,
        lastTimestamp: response.data?.lastTimestamp
      }
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.hasNextPage ? lastPage.lastTimestamp : undefined,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!sourceType && !!sourceId
  })
}



export const useStageEvents = (canvasId: string, stageId: string, states: SuperplaneStageEventState[]) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.stageEvents(canvasId, stageId, states),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListStageEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, stageIdOrName: stageId },
          query: {
            states: states,
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      )
      return {
        events: response.data?.events || [],
        totalCount: response.data?.totalCount || 0,
        hasNextPage: response.data?.hasNextPage || false,
        lastTimestamp: response.data?.lastTimestamp
      }
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.hasNextPage ? lastPage.lastTimestamp : undefined,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!stageId
  })
}

export const useStageExecutions = (canvasId: string, stageId: string, results?: string[]) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.stageExecutions(canvasId, stageId, results),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListStageExecutions(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, stageIdOrName: stageId },
          query: {
            results: results,
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      )
      return {
        executions: response.data?.executions || [],
        totalCount: response.data?.totalCount || 0,
        hasNextPage: response.data?.hasNextPage || false,
        lastTimestamp: response.data?.lastTimestamp
      }
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.hasNextPage ? lastPage.lastTimestamp : undefined,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!stageId
  })
}

export const useEventRejections = (canvasId: string, targetType: string, targetId: string) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.eventRejections(canvasId, targetType, targetId),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListEventRejections(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          query: {
            targetType,
            targetId,
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      )
      return {
        rejections: response.data?.rejections || [],
        totalCount: response.data?.totalCount || 0,
        hasNextPage: response.data?.hasNextPage || false,
        lastTimestamp: response.data?.lastTimestamp
      }
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.hasNextPage ? lastPage.lastTimestamp : undefined,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!targetType && !!targetId
  })
}


export const useAlertsBySourceId = (canvasId: string) => {
  return useQuery({
    queryKey: canvasKeys.alerts(canvasId),
    queryFn: async () => {
      const response = await superplaneListAlerts(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
        })
      )

      const alertsBySourceId = response.data?.alerts?.reduce((acc, alert) => {
        if (!alert.sourceId) return acc

        if (!acc[alert.sourceId]) {
          acc[alert.sourceId] = []
        }
        acc[alert.sourceId].push(alert)
        return acc
      }, {} as Record<string, SuperplaneAlert[]>) || {}
      return alertsBySourceId
    },
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId
  })
}

// Alert-related hooks
export const useAcknowledgeAlert = (canvasId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (alertId: string) => {
      return await superplaneAcknowledgeAlert(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, alertId }
        })
      )
    },
    onMutate: async (alertId) => {
      await queryClient.cancelQueries({ queryKey: canvasKeys.alerts(canvasId) })

      const previousAlerts = queryClient.getQueryData<Record<string, SuperplaneAlert[]>>(canvasKeys.alerts(canvasId))

      if (previousAlerts) {
        const updatedAlerts = { ...previousAlerts }

        for (const sourceId in updatedAlerts) {
          updatedAlerts[sourceId] = updatedAlerts[sourceId].filter(alert => alert.id !== alertId)

          if (updatedAlerts[sourceId].length === 0) {
            delete updatedAlerts[sourceId]
          }
        }

        queryClient.setQueryData(canvasKeys.alerts(canvasId), updatedAlerts)
      }

      return { previousAlerts }
    },
    onError: (_err, _alertId, context) => {
      if (context?.previousAlerts) {
        queryClient.setQueryData(canvasKeys.alerts(canvasId), context.previousAlerts)
      }
    }
  })
}

export const useAddAlert = (canvasId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (alert: SuperplaneAlert) => {
      return { alert }
    },
    onMutate: async (newAlert) => {
      await queryClient.cancelQueries({ queryKey: canvasKeys.alerts(canvasId) })

      const previousAlerts = queryClient.getQueryData<Record<string, SuperplaneAlert[]>>(canvasKeys.alerts(canvasId))

      if (newAlert.sourceId) {
        const updatedAlerts = { ...(previousAlerts || {}) }

        if (!updatedAlerts[newAlert.sourceId]) {
          updatedAlerts[newAlert.sourceId] = []
        }

        updatedAlerts[newAlert.sourceId] = [newAlert, ...updatedAlerts[newAlert.sourceId]]

        queryClient.setQueryData(canvasKeys.alerts(canvasId), updatedAlerts)
      }

      return { previousAlerts }
    },
    onError: (_err, _newAlert, context) => {
      if (context?.previousAlerts) {
        queryClient.setQueryData(canvasKeys.alerts(canvasId), context.previousAlerts)
      }
    }
  })
}

// Canvas Role Management Hooks
export const useCanvasRole = (canvasId: string, roleName: string) => {
  return useQuery({
    queryKey: canvasKeys.role(canvasId, roleName),
    queryFn: async () => {
      const response = await rolesDescribeRole(
        withOrganizationHeader({
          path: {
            roleName,
          },
          query: {
            domainType: 'DOMAIN_TYPE_CANVAS',
            domainId: canvasId,
          }
        })
      )
      return response.data?.role || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId && !!roleName,
  })
}

export const useCreateCanvasRole = (canvasId: string) => {
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
      // Invalidate and refetch canvas roles
      queryClient.invalidateQueries({ queryKey: canvasKeys.roles(canvasId) })
    }
  })
}

export const useUpdateCanvasRole = (canvasId: string) => {
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
      queryClient.invalidateQueries({ queryKey: canvasKeys.roles(canvasId) })
      queryClient.invalidateQueries({ queryKey: canvasKeys.role(canvasId, variables.roleName) })
    }
  })
}

export const useDeleteCanvasRole = (canvasId: string) => {
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
      queryClient.invalidateQueries({ queryKey: canvasKeys.roles(canvasId) })
    }
  })
}
