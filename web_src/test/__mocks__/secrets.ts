import { IntegrationsIntegration, SecretsSecret } from "../../src/api-client"

export const createMockSecrets = (): SecretsSecret[] => [
  {
    metadata: { id: '1', name: 'secret-1' },
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
    metadata: { id: '2', name: 'secret-2' },
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
    metadata: { id: '3', name: 'secret-3' },
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

export const createMockIntegrations = (): IntegrationsIntegration[] => [
  { metadata: { name: 'existing-github-integration' } },
  { metadata: { name: 'production-semaphore' } }
]

export const defaultProps = {
  organizationId: 'org-123',
  canvasId: 'canvas-456'
}