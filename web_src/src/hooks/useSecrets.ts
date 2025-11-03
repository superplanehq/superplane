import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  secretsListSecrets,
  secretsCreateSecret,
  secretsDescribeSecret,
  secretsUpdateSecret,
  secretsDeleteSecret,
} from '@/api-client/sdk.gen'
import { withOrganizationHeader } from '@/utils/withOrganizationHeader'
import type { SecretsCreateSecretData, SecretsUpdateSecretData } from '@/api-client/types.gen'

export const secretKeys = {
  all: ['secrets'] as const,
  byDomain: (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => [...secretKeys.all, 'domain', domainId, domainType] as const,
  detail: (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", secretId: string) => [...secretKeys.byDomain(domainId, domainType), 'detail', secretId] as const,
}

export const useSecrets = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => {
  return useQuery({
    queryKey: secretKeys.byDomain(domainId, domainType),
    queryFn: async () => {
      const response = await secretsListSecrets(withOrganizationHeader({
        query: { domainId: domainId, domainType: domainType },
      }))
      return response.data?.secrets || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!domainId,
  })
}

export const useSecret = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", secretId: string) => {
  return useQuery({
    queryKey: secretKeys.detail(domainId, domainType, secretId),
    queryFn: async () => {
      const response = await secretsDescribeSecret(withOrganizationHeader({
        query: {
          domainType: domainType,
          domainId: domainId,
        },
        path: { idOrName: secretId }
      }))
      return response.data?.secret || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!domainId && !!secretId,
  })
}

export interface CreateSecretParams {
  name: string
  environmentVariables: Array<{ name: string; value: string }>
}

export const useCreateSecret = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: CreateSecretParams) => {
      const data: { [key: string]: string } = {}
      params.environmentVariables.forEach(env => {
        data[env.name] = env.value
      })

      const secret: SecretsCreateSecretData['body'] = {
        secret: {
          metadata: {
            name: params.name,
            id: '',
            domainId: domainId,
            domainType: domainType,
            createdAt: new Date().toISOString(),
          },
          spec: {
            provider: 'PROVIDER_LOCAL',
            local: {
              data
            }
          }
        },
        domainId,
        domainType
      }

      return await secretsCreateSecret(withOrganizationHeader({
        body: secret
      }))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: secretKeys.byDomain(domainId, domainType)
      })
    }
  })
}

export interface UpdateSecretParams {
  name: string
  environmentVariables: Array<{ name: string; value: string }>
}

export const useUpdateSecret = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION", secretId: string) => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: UpdateSecretParams) => {
      const data: { [key: string]: string } = {}
      params.environmentVariables.forEach(env => {
        data[env.name] = env.value
      })

      const secret: SecretsUpdateSecretData['body'] = {
        secret: {
          metadata: {
            name: params.name,
            id: secretId,
            domainId: domainId,
            domainType: domainType,
            createdAt: new Date().toISOString(),
        },
        spec: {
          provider: 'PROVIDER_LOCAL',
          local: {
            data
          }
        }
      },
      domainId,
      domainType
    }

      return await secretsUpdateSecret(withOrganizationHeader({
        body: {
          secret: secret.secret,
          domainId,
          domainType,
        },
        path: {
          idOrName: secretId
        }
      }))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: secretKeys.byDomain(domainId, domainType)
      })
      queryClient.invalidateQueries({
        queryKey: secretKeys.detail(domainId, domainType, secretId)
      })
    }
  })
}

export const useDeleteSecret = (domainId: string, domainType: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION") => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (secretId: string) => {
      return await secretsDeleteSecret(withOrganizationHeader({
        path: {
          idOrName: secretId
        },
        query: {
          domainId: domainId,
          domainType: domainType
        }
      }))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: secretKeys.byDomain(domainId, domainType)
      })
    }
  })
}
