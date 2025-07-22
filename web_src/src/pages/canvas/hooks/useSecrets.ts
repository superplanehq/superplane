import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  superplaneListSecrets,
  superplaneCreateSecret,
  superplaneDescribeSecret,
  superplaneUpdateSecret,
  superplaneDeleteSecret,
} from '../../../api-client/sdk.gen'
import type { SuperplaneCreateSecretData, SuperplaneSecret } from '../../../api-client/types.gen'

export const secretKeys = {
  all: ['secrets'] as const,
  byCanvas: (canvasId: string) => [...secretKeys.all, 'canvas', canvasId] as const,
  detail: (canvasId: string, secretId: string) => [...secretKeys.byCanvas(canvasId), 'detail', secretId] as const,
}

export const useSecrets = (canvasId: string) => {
  return useQuery({
    queryKey: secretKeys.byCanvas(canvasId),
    queryFn: async () => {
      const response = await superplaneListSecrets({
        path: { canvasIdOrName: canvasId }
      })
      return response.data?.secrets || []
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId,
  })
}

export const useSecret = (canvasId: string, secretId: string) => {
  return useQuery({
    queryKey: secretKeys.detail(canvasId, secretId),
    queryFn: async () => {
      const response = await superplaneDescribeSecret({
        path: { 
          canvasIdOrName: canvasId,
          idOrName: secretId 
        }
      })
      return response.data?.secret || null
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!canvasId && !!secretId,
  })
}

export interface CreateSecretParams {
  name: string
  environmentVariables: Array<{ name: string; value: string }>
}

export const useCreateSecret = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: CreateSecretParams) => {
      const data: { [key: string]: string } = {}
      params.environmentVariables.forEach(env => {
        data[env.name] = env.value
      })

      const secret: SuperplaneCreateSecretData['body'] = {
        secret: {
          metadata: {
            name: params.name,
            id: '',
            canvasId: canvasId,
            createdAt: new Date().toISOString(),
          },
          spec: {
            provider: 'PROVIDER_LOCAL',
            local: {
              data
            }
          }
        }
      }

      return await superplaneCreateSecret({
        path: { canvasIdOrName: canvasId },
        body: secret
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: secretKeys.byCanvas(canvasId) 
      })
    }
  })
}

export interface UpdateSecretParams {
  name: string
  environmentVariables: Array<{ name: string; value: string }>
}

export const useUpdateSecret = (canvasId: string, secretId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (params: UpdateSecretParams) => {
      const data: { [key: string]: string } = {}
      params.environmentVariables.forEach(env => {
        data[env.name] = env.value
      })

      const secret: SuperplaneSecret = {
        metadata: {
          name: params.name,
          id: secretId,
          canvasId: canvasId,
          createdAt: new Date().toISOString(),
        },
        spec: {
          provider: 'PROVIDER_LOCAL',
          local: {
            data
          }
        }
      }

      return await superplaneUpdateSecret({
        path: { 
          canvasIdOrName: canvasId,
          idOrName: secretId
        },
        body: { secret }
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: secretKeys.byCanvas(canvasId) 
      })
      queryClient.invalidateQueries({ 
        queryKey: secretKeys.detail(canvasId, secretId) 
      })
    }
  })
}

export const useDeleteSecret = (canvasId: string) => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (secretId: string) => {
      return await superplaneDeleteSecret({
        path: { 
          canvasIdOrName: canvasId,
          idOrName: secretId
        }
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: secretKeys.byCanvas(canvasId) 
      })
    }
  })
}