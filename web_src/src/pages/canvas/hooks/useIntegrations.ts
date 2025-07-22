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

export const useCreateIntegration = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: CreateIntegrationParams) => {
      const integration: SuperplaneCreateIntegrationData['body'] = {
        integration: {
          metadata: {
            name: params.name,
            id: '',
            createdBy: '',
            createdAt: '',
            domainType: 'DOMAIN_TYPE_CANVAS',
            domainId: canvasId,
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