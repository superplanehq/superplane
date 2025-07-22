import { useState } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import { useIntegrations, useCreateIntegration, type CreateIntegrationParams } from '../../../pages/canvas/hooks/useIntegrations'

interface CanvasIntegrationsProps {
  canvasId: string
  organizationId: string
}

type IntegrationSection = 'list' | 'choose-type' | 'new'
type IntegrationType = 'TYPE_SEMAPHORE' | 'TYPE_GITHUB'

const INTEGRATION_TYPES = [
  {
    value: 'TYPE_SEMAPHORE' as const,
    label: 'Semaphore',
    description: 'Connect to Semaphore CI/CD pipelines for automated deployments and testing workflows',
    icon: 'build_circle',
    color: 'bg-orange-100 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400',
    popular: true
  },
  {
    value: 'TYPE_GITHUB' as const,
    label: 'GitHub',
    description: 'Integrate with GitHub repositories for webhook events, deployments, and actions',
    icon: 'code',
    color: 'bg-gray-100 dark:bg-gray-900/30 text-gray-600 dark:text-gray-400',
    popular: true
  }
]

export function CanvasIntegrations({ canvasId }: CanvasIntegrationsProps) {
  const [section, setSection] = useState<IntegrationSection>('list')
  const [selectedType, setSelectedType] = useState<IntegrationType>('TYPE_SEMAPHORE')
  const [integrationName, setIntegrationName] = useState('')
  const [integrationUrl, setIntegrationUrl] = useState('')
  const [authType, setAuthType] = useState<'AUTH_TYPE_TOKEN' | 'AUTH_TYPE_OIDC' | 'AUTH_TYPE_NONE'>('AUTH_TYPE_NONE')
  const [tokenSecretName, setTokenSecretName] = useState('')
  const [oidcEnabled, setOidcEnabled] = useState(false)
  
  const { data: integrations, isLoading, error } = useIntegrations(canvasId)
  const createIntegrationMutation = useCreateIntegration(canvasId)

  const handleAddIntegration = () => {
    setSection('choose-type')
  }

  const handleTypeSelection = (type: IntegrationType) => {
    setSelectedType(type)
    setSection('new')
    // Set default URL placeholder based on type
    if (type === 'TYPE_GITHUB') {
      setIntegrationUrl('')
    } else {
      setIntegrationUrl('')
    }
  }

  const resetForm = () => {
    setIntegrationName('')
    setIntegrationUrl('')
    setAuthType('AUTH_TYPE_NONE')
    setTokenSecretName('')
    setOidcEnabled(false)
  }

  const handleCreateIntegration = async () => {
    if (!integrationName.trim() || !integrationUrl.trim()) {
      return
    }

    const integrationData: CreateIntegrationParams = {
      name: integrationName.trim(),
      type: selectedType,
      url: integrationUrl.trim(),
      authType,
      tokenSecretName: authType === 'AUTH_TYPE_TOKEN' ? tokenSecretName.trim() : undefined,
      oidcEnabled: authType === 'AUTH_TYPE_OIDC' ? oidcEnabled : undefined,
    }

    try {
      await createIntegrationMutation.mutateAsync(integrationData)
      resetForm()
      setSection('list')
    } catch (error) {
      console.error('Failed to create integration:', error)
    }
  }

  const hasIntegrations = integrations && integrations.length > 0

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
        {section === 'choose-type' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">Choose integration type</span>
          </>
        )}
        {section === 'new' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <button 
              onClick={() => setSection('choose-type')}
              className="hover:text-zinc-900 dark:hover:text-zinc-100"
            >
              Choose integration type
            </button>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">
              New {INTEGRATION_TYPES.find(t => t.value === selectedType)?.label} integration
            </span>
          </>
        )}
      </div>

      {/* List Section */}
      {section === 'list' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>Integrations</Heading>
            {hasIntegrations && (
              <Button color="blue" onClick={handleAddIntegration}>
                <MaterialSymbol name="add" size="sm" />
                Add Integration
              </Button>
            )}
          </div>
          
          {isLoading ? (
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="text-center py-12">
                <MaterialSymbol name="progress_activity" className="mx-auto text-zinc-400 mb-4 animate-spin" size="xl" />
                <Text className="text-zinc-600 dark:text-zinc-400">Loading integrations...</Text>
              </div>
            </div>
          ) : error ? (
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="text-center py-12">
                <MaterialSymbol name="error" className="mx-auto text-red-400 mb-4" size="xl" />
                <Heading level={3} className="text-lg mb-2">Failed to load integrations</Heading>
                <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
                  There was an error loading your integrations. Please try again.
                </Text>
              </div>
            </div>
          ) : !hasIntegrations ? (
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="text-center py-12">
                <MaterialSymbol name="integration_instructions" className="mx-auto text-zinc-400 mb-4" size="xl" />
                <Heading level={3} className="text-lg mb-2">No integrations configured</Heading>
                <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
                  Connect external services like Semaphore, GitHub, and other tools to enhance your canvas workflows.
                </Text>
                <Button color="blue" onClick={handleAddIntegration}>
                  <MaterialSymbol name="add" size="sm" />
                  Add Integration
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {integrations.map((integration) => (
                <div
                  key={integration.metadata?.id}
                  className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-4">
                      <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                        <MaterialSymbol 
                          name={integration.spec?.type === 'TYPE_GITHUB' ? 'code' : 'build_circle'} 
                          className="text-blue-600 dark:text-blue-400" 
                          size="lg" 
                        />
                      </div>
                      <div>
                        <Heading level={3} className="text-lg mb-1">
                          {integration.metadata?.name}
                        </Heading>
                        <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-2">
                          {integration.spec?.type === 'TYPE_GITHUB' ? 'GitHub' : 
                           integration.spec?.type === 'TYPE_SEMAPHORE' ? 'Semaphore' : 
                           'Unknown'} Integration
                        </Text>
                        {integration.spec?.url && (
                          <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                            {integration.spec.url}
                          </Text>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <div className={`px-2 py-1 rounded-full text-xs font-medium ${
                        integration.spec?.auth?.use === 'AUTH_TYPE_TOKEN' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' :
                        integration.spec?.auth?.use === 'AUTH_TYPE_OIDC' ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400' :
                        'bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400'
                      }`}>
                        {integration.spec?.auth?.use === 'AUTH_TYPE_TOKEN' ? 'Token Auth' :
                         integration.spec?.auth?.use === 'AUTH_TYPE_OIDC' ? 'OIDC Auth' :
                         'No Auth'}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      {/* Choose Type Section */}
      {section === 'choose-type' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>Choose integration type</Heading>
          </div>

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
              Select the type of integration you want to create. Each integration type provides different capabilities for your workflows.
            </Text>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {INTEGRATION_TYPES.map((integrationType) => (
                <button
                  key={integrationType.value}
                  onClick={() => handleTypeSelection(integrationType.value)}
                  className="relative flex items-start gap-4 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg hover:border-blue-300 dark:hover:border-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors text-left group"
                >
                  <div className={`p-3 rounded-lg ${integrationType.color}`}>
                    <MaterialSymbol name={integrationType.icon} size="lg" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <Heading level={3} className="text-lg group-hover:text-blue-600 dark:group-hover:text-blue-400">
                        {integrationType.label}
                      </Heading>
                      {integrationType.popular && (
                        <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 text-xs font-medium rounded-full">
                          Popular
                        </span>
                      )}
                    </div>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      {integrationType.description}
                    </Text>
                  </div>
                  <MaterialSymbol 
                    name="arrow_forward" 
                    className="text-zinc-400 group-hover:text-blue-500 dark:group-hover:text-blue-400 transition-colors"
                    size="sm" 
                  />
                </button>
              ))}
            </div>

            <div className="flex gap-3 mt-6">
              <Button onClick={() => setSection('list')}>
                <MaterialSymbol name="arrow_back" size="sm" />
                Back
              </Button>
            </div>
          </div>
        </>
      )}

      {/* New Integration Section */}
      {section === 'new' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>
              New {INTEGRATION_TYPES.find(t => t.value === selectedType)?.label} integration
            </Heading>
          </div>

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <div className="space-y-6">
              {/* Integration Name */}
              <div>
                <label htmlFor="integration-name" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  Integration Name
                </label>
                <Input
                  id="integration-name"
                  type="text"
                  placeholder={`Enter a name for this ${INTEGRATION_TYPES.find(t => t.value === selectedType)?.label} integration`}
                  value={integrationName}
                  onChange={(e) => setIntegrationName(e.target.value)}
                  className="w-full"
                />
              </div>

              {/* URL */}
              <div>
                <label htmlFor="integration-url" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  URL
                </label>
                <Input
                  id="integration-url"
                  type="url"
                  placeholder={
                    selectedType === 'TYPE_GITHUB' 
                      ? 'https://github.com/owner/repo' 
                      : 'https://api.semaphoreci.com'
                  }
                  value={integrationUrl}
                  onChange={(e) => setIntegrationUrl(e.target.value)}
                  className="w-full"
                />
              </div>

              {/* Authentication */}
              <div>
                <label htmlFor="auth-type" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  Authentication
                </label>
                <select
                  id="auth-type"
                  value={authType}
                  onChange={(e) => setAuthType(e.target.value as typeof authType)}
                  className="mt-2 block w-full rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="AUTH_TYPE_NONE">No Authentication</option>
                  <option value="AUTH_TYPE_TOKEN">Token Authentication</option>
                  <option value="AUTH_TYPE_OIDC">OIDC Authentication</option>
                </select>
              </div>

              {/* Token Secret Name (if token auth) */}
              {authType === 'AUTH_TYPE_TOKEN' && (
                <div>
                  <label htmlFor="token-secret" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                    Token Secret Name
                  </label>
                  <Input
                    id="token-secret"
                    type="text"
                    placeholder="Name of the secret containing the auth token"
                    value={tokenSecretName}
                    onChange={(e) => setTokenSecretName(e.target.value)}
                    className="w-full"
                  />
                  <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                    Reference a secret that contains the authentication token for this integration.
                  </Text>
                </div>
              )}

              {/* OIDC Enabled (if OIDC auth) */}
              {authType === 'AUTH_TYPE_OIDC' && (
                <div>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={oidcEnabled}
                      onChange={(e) => setOidcEnabled(e.target.checked)}
                      className="rounded border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-800 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="text-sm text-zinc-700 dark:text-zinc-300">Enable OIDC Authentication</span>
                  </label>
                </div>
              )}

              {/* Action buttons */}
              <div className="flex gap-3">
                <Button onClick={() => setSection('choose-type')}>
                  <MaterialSymbol name="arrow_back" size="sm" />
                  Back
                </Button>
                <Button 
                  color="blue" 
                  onClick={handleCreateIntegration}
                  disabled={createIntegrationMutation.isPending || !integrationName.trim() || !integrationUrl.trim()}
                >
                  {createIntegrationMutation.isPending ? (
                    <>
                      <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <MaterialSymbol name="add" size="sm" />
                      Create Integration
                    </>
                  )}
                </Button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}