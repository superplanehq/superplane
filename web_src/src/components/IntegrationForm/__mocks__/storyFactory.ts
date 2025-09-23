import { IntegrationsIntegration, SecretsSecret } from "@/api-client"

export const createMockSecrets = (): SecretsSecret[] => [
  {
    metadata: { id: '1', name: 'github-pat' },
    spec: {
      local: {
        data: {
          'api-token': 'ghp_xxxxxxxxxxxxxxxxxxxx',
          'backup-token': 'ghp_yyyyyyyyyyyyyyyyyyyy',
          'webhook-secret': 'whs_zzzzzzzzzzzzzzzzzzzz'
        }
      }
    }
  },
  {
    metadata: { id: '2', name: 'semaphore-key' },
    spec: {
      local: {
        data: {
          'token': 'smp_aaaaaaaaaaaaaaaaaaa',
          'api-key': 'smp_bbbbbbbbbbbbbbbbbb'
        }
      }
    }
  },
  {
    metadata: { id: '3', name: 'service-keys' },
    spec: {
      local: {
        data: {
          'primary': 'key_111111111111111',
          'secondary': 'key_222222222222222',
          'fallback': 'key_333333333333333'
        }
      }
    }
  }
]

export const createGitHubMockSecrets = (): SecretsSecret[] => [
  {
    metadata: { id: '1', name: 'github-pat-1' },
    spec: {
      local: {
        data: {
          'api-token': 'ghp_xxxxxxxxxxxxxxxxxxxx',
          'backup-token': 'ghp_yyyyyyyyyyyyyyyyyyyy'
        }
      }
    }
  },
  {
    metadata: { id: '2', name: 'my-github-secret' },
    spec: {
      local: {
        data: {
          'token': 'ghp_zzzzzzzzzzzzzzzzzzzz'
        }
      }
    }
  }
]

export const createSemaphoreMockSecrets = (): SecretsSecret[] => [
  {
    metadata: { id: '1', name: 'semaphore-api-key' },
    spec: {
      local: {
        data: {
          'api-token': 'smp_xxxxxxxxxxxxxxxxxxxx',
          'backup-token': 'smp_yyyyyyyyyyyyyyyyyyyy'
        }
      }
    }
  },
  {
    metadata: { id: '2', name: 'my-semaphore-secret' },
    spec: {
      local: {
        data: {
          'token': 'smp_zzzzzzzzzzzzzzzzzzzz'
        }
      }
    }
  }
]

export const createFlowMockSecrets = (): SecretsSecret[] => [
  {
    metadata: { id: '1', name: 'github-pat-production' },
    spec: {
      local: {
        data: {
          'api-token': 'ghp_xxxxxxxxxxxxxxxxxxxx',
          'webhook-secret': 'whs_yyyyyyyyyyyyyyyyyyyy'
        }
      }
    }
  },
  {
    metadata: { id: '2', name: 'semaphore-api-key' },
    spec: {
      local: {
        data: {
          'token': 'smp_zzzzzzzzzzzzzzzzzzzz',
          'backup-key': 'smp_aaaaaaaaaaaaaaaaaaa'
        }
      }
    }
  }
]

export const createMockIntegrations = (): IntegrationsIntegration[] => [
  { metadata: { name: 'existing-github-integration' } },
  { metadata: { name: 'production-semaphore' } }
]

export const defaultProps = {
  organizationId: 'org-123',
  canvasId: 'canvas-456'
}