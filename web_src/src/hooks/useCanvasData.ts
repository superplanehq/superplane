import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query'
import {
  rolesListRoles,
  usersListUsers,
  rolesAssignRole,
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
  superplaneBulkListEvents,
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
  events: (canvasId: string, sourceType: string, sourceId: string) => [...canvasKeys.all, 'events', canvasId, sourceType, sourceId] as const,
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
export const useEventSourceEvents = (canvasId: string, eventSourceId: string) => {
  return useInfiniteQuery({
    queryKey: canvasKeys.events(canvasId, 'EVENT_SOURCE_TYPE_EVENT_SOURCE', eventSourceId),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          query: {
            sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE',
            sourceId: eventSourceId,
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      )
      return {
        events: response.data?.events || [],
        nextCursor: response.data?.events && response.data.events.length === 20 
          ? response.data.events[response.data.events.length - 1]?.receivedAt 
          : undefined
      }
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!eventSourceId
  })
}

export const useConnectedSourcesEvents = (
  canvasId: string, 
  connectedSources: {
    eventSourceIds: string[];
    stageIds: string[];
    connectionGroupIds: string[];
  },
  limitPerSource: number = 20
) => {
  return useInfiniteQuery({
    queryKey: ['connectedEvents', canvasId, connectedSources, limitPerSource],
    queryFn: async ({ pageParam }: { pageParam: string | undefined }) => {
      const sources = [
        ...connectedSources.eventSourceIds.map(id => ({
          sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE' as const,
          sourceId: id
        })),
        ...connectedSources.stageIds.map(id => ({
          sourceType: 'EVENT_SOURCE_TYPE_STAGE' as const,
          sourceId: id
        })),
        ...connectedSources.connectionGroupIds.map(id => ({
          sourceType: 'EVENT_SOURCE_TYPE_CONNECTION_GROUP' as const,
          sourceId: id
        }))
      ];

      if (sources.length === 0) {
        return { 
          events: [], 
          nextCursor: undefined, 
          limitPerSource 
        };
      }

      const response = await superplaneBulkListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
          body: {
            sources,
            limitPerSource: limitPerSource,
            before: pageParam 
          }
        })
      );

      const allEvents = response.data?.results?.flatMap(result => result.events || []) || [];

      allEvents.sort((a, b) => 
        new Date(b.receivedAt || '').getTime() - new Date(a.receivedAt || '').getTime()
      );

      const nextCursor = allEvents.length > 0 && allEvents.length >= limitPerSource 
        ? allEvents[allEvents.length - 1].receivedAt 
        : undefined;

      return {
        events: allEvents,
        limitPerSource,
        nextCursor
      };
    },
    getNextPageParam: (lastPage): string | undefined => lastPage.nextCursor,
    initialPageParam: undefined as string | undefined,
    enabled: Object.values(connectedSources).some(arr => arr.length > 0)
  });
};
