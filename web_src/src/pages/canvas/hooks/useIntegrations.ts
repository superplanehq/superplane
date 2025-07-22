import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  superplaneListIntegrations,
  superplaneCreateIntegration,
  superplaneDescribeIntegration,
} from '../../../api-client/sdk.gen'
import type { SuperplaneCreateIntegrationData } from '../../../api-client/types.gen'

export const integrationKeys = {
  all: ['integrations'] as const,
  byCanvas: (canvasId: string) => [...integrationKeys.all, 'canvas', canvasId] as const,
  detail: (canvasId: string, integrationId: string) => [...integrationKeys.byCanvas(canvasId), 'detail', integrationId] as const,
}

export const useIntegrations = (canvasId: string) => {
  return useQuery({
    queryKey: integrationKeys.byCanvas(canvasId),
    queryFn: async () => {
      const response = await superplaneListIntegrations({
        path: { canvasIdOrName: canvasId }
      })
      return response.data?.integrations || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId,
  })
}

export const useIntegration = (canvasId: string, integrationId: string) => {
  return useQuery({
    queryKey: integrationKeys.detail(canvasId, integrationId),
    queryFn: async () => {
      const response = await superplaneDescribeIntegration({
        path: { 
          canvasIdOrName: canvasId,
          idOrName: integrationId 
        }
      })
      return response.data?.integration || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId && !!integrationId,
  })
}

export interface CreateIntegrationParams {
  name: string
  type: 'TYPE_SEMAPHORE' | 'TYPE_GITHUB'
  url: string
  authType: 'AUTH_TYPE_TOKEN' | 'AUTH_TYPE_OIDC' | 'AUTH_TYPE_NONE'
  tokenSecretName?: string
  oidcEnabled?: boolean
}

export interface UpdateIntegrationParams extends CreateIntegrationParams {
  id: string
}

export const useCreateIntegration = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: CreateIntegrationParams) => {
      const integration: SuperplaneCreateIntegrationData['body'] = {
        integration: {
          metadata: {
            name: params.name
          },
          spec: {
            type: params.type,
            url: params.url,
            auth: {
              use: params.authType,
              ...(params.authType === 'AUTH_TYPE_TOKEN' && params.tokenSecretName && {
                token: {
                  valueFrom: {
                    secret: {
                      name: params.tokenSecretName,
                      key: 'token'
                    }
                  }
                }
              })
            },
            oidc: {
              enabled: params.oidcEnabled || false
            }
          }
        }
      }

      return await superplaneCreateIntegration({
        path: { canvasIdOrName: canvasId },
        body: integration
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byCanvas(canvasId) 
      })
    }
  })
}

export const useUpdateIntegration = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: UpdateIntegrationParams) => {
      // Mock update integration - in real implementation this would call an update API
      console.log('Updating integration:', params)
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1000))
      return { success: true, integrationId: params.id }
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byCanvas(canvasId) 
      })
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.detail(canvasId, data.integrationId) 
      })
    }
  })
}

export const useDeleteIntegration = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (integrationId: string) => {
      // Mock delete integration - in real implementation this would call a delete API
      console.log('Deleting integration:', integrationId)
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 500))
      return { success: true, integrationId }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byCanvas(canvasId) 
      })
    }
  })
}