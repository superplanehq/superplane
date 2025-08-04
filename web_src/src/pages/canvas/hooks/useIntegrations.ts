import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  integrationsListIntegrations,
  integrationsCreateIntegration,
  integrationsDescribeIntegration,
} from '../../../api-client/sdk.gen'
import type { IntegrationsCreateIntegrationData } from '../../../api-client/types.gen'

export const integrationKeys = {
  all: ['integrations'] as const,
  byDomain: (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => [...integrationKeys.all, 'domain', domainId, domainType] as const,
  detail: (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", integrationId: string) => [...integrationKeys.byDomain(domainId, domainType), 'detail', integrationId] as const,
}

export const useIntegrations = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => {
  return useQuery({
    queryKey: integrationKeys.byDomain(domainId, domainType),
    queryFn: async () => {
      const response = await integrationsListIntegrations({
        query: { domainId: domainId, domainType: domainType }
      })
      return response.data?.integrations || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!domainId && !!domainType,
  })
}

export const useIntegration = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", integrationId: string) => {
  return useQuery({
    queryKey: integrationKeys.detail(domainId, domainType, integrationId),
    queryFn: async () => {
      const response = await integrationsDescribeIntegration({
        query: {
          domainId: domainId,
          domainType: domainType,
        },
        path: { 
          idOrName: integrationId 
        }
      })
      return response.data?.integration || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!domainId && !!domainType && !!integrationId,
  })
}

export interface CreateIntegrationParams {
  name: string
  type: string
  url: string
  authType: 'AUTH_TYPE_TOKEN' | 'AUTH_TYPE_OIDC' | 'AUTH_TYPE_NONE'
  tokenSecretName?: string
  tokenSecretKey?: string
}

export interface UpdateIntegrationParams extends CreateIntegrationParams {
  id: string
}

export const useCreateIntegration = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: CreateIntegrationParams) => {
      const integration: IntegrationsCreateIntegrationData['body'] = {
        domainId,
        domainType,
        integration: {
          metadata: {
            name: params.name,
            domainId: domainId,
            domainType: domainType,
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
                      key: params.tokenSecretKey || 'token'
                    }
                  }
                }
              })
            }
          }
        }
      }

      return await integrationsCreateIntegration({
        body: integration
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byDomain(domainId, domainType) 
      })
    }
  })
}

export const useUpdateIntegration = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", _integrationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: UpdateIntegrationParams) => {
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1000))
      return { success: true, integrationId: params.id }
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byDomain(domainId, domainType) 
      })
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.detail(domainId, domainType, data.integrationId) 
      })
    }
  })
}

export const useDeleteIntegration = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", integrationId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async () => {
      await new Promise(resolve => setTimeout(resolve, 500))
      return { success: true, integrationId }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: integrationKeys.byDomain(domainId, domainType) 
      })
    }
  })
}