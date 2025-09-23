import { useState } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import { useIntegrations, useCreateIntegration, useUpdateIntegration, type CreateIntegrationParams, type UpdateIntegrationParams } from '../../../pages/canvas/hooks/useIntegrations'
import { useSecrets, useSecret } from '../../../pages/canvas/hooks/useSecrets'
import { IntegrationsIntegration } from '@/api-client'
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg'
import GithubLogo from '@/assets/github-mark.svg'
import { useNavigate } from 'react-router-dom'
import { showErrorToast, showSuccessToast } from '../../../utils/toast'

interface CanvasIntegrationsProps {
  canvasId: string
  organizationId: string
}

type IntegrationSection = 'list' | 'choose-type' | 'new' | 'edit'

interface IntegrationType {
  value: string
  label: string
  description: string
  icon: React.ReactNode | string
  color: string
  popular: boolean
}

const INTEGRATION_TYPES: Record<string, IntegrationType> = {
  'semaphore': {
    value: 'semaphore' as const,
    label: 'Semaphore',
    description: 'Connect to Semaphore CI/CD pipelines for automated deployments and testing workflows',
    icon: SemaphoreLogo,
    color: 'bg-gray-100 dark:bg-gray-900/30 text-gray-600 dark:text-gray-400',
    popular: true
  },
  'github': {
    value: 'github' as const,
    label: 'GitHub',
    description: 'Connect to GitHub repositories for Actions running workflows',
    icon: GithubLogo,
    color: 'bg-gray-100 dark:bg-gray-900/30 text-gray-600 dark:text-gray-400',
    popular: true
  }
}

export function CanvasIntegrations({ canvasId, organizationId }: CanvasIntegrationsProps) {
  const [section, setSection] = useState<IntegrationSection>('list')
  const [selectedType, setSelectedType] = useState<string>('semaphore')
  const [integrationName, setIntegrationName] = useState('')
  const [integrationUrl, setIntegrationUrl] = useState('')
  const [authType, setAuthType] = useState<'AUTH_TYPE_TOKEN' | 'AUTH_TYPE_OIDC' | 'AUTH_TYPE_NONE'>('AUTH_TYPE_NONE')
  const [selectedSecretId, setSelectedSecretId] = useState('')
  const [selectedSecretKey, setSelectedSecretKey] = useState('')
  const [editingIntegration, setEditingIntegration] = useState<IntegrationsIntegration | null>(null)

  const { data: integrations, isLoading, error } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS")
  const createIntegrationMutation = useCreateIntegration(canvasId, "DOMAIN_TYPE_CANVAS")
  const updateIntegrationMutation = useUpdateIntegration(canvasId, "DOMAIN_TYPE_CANVAS", editingIntegration?.metadata?.id || '')
  const { data: secrets = [] } = useSecrets(canvasId, "DOMAIN_TYPE_CANVAS")
  const { data: selectedSecret } = useSecret(canvasId, "DOMAIN_TYPE_CANVAS", selectedSecretId)

  const navigate = useNavigate()

  const handleAddIntegration = () => {
    setSection('choose-type')
  }

  const handleTypeSelection = (type: string) => {
    setSelectedType(type)
    setSection('new')
    setIntegrationUrl('')
  }

  const resetForm = () => {
    setIntegrationName('')
    setIntegrationUrl('')
    setAuthType('AUTH_TYPE_NONE')
    setSelectedSecretId('')
    setSelectedSecretKey('')
    setEditingIntegration(null)
  }

  const handleCreateIntegration = async () => {
    const trimmedIntegrationName = integrationName.trim()
    let trimmedIntegrationUrl = integrationUrl.trim()

    if (!trimmedIntegrationName || !trimmedIntegrationUrl) {
      return
    }

    if (trimmedIntegrationUrl.endsWith('/')) {
      trimmedIntegrationUrl = trimmedIntegrationUrl.slice(0, -1)
    }


    // Validate token authentication requirements
    if (authType === 'AUTH_TYPE_TOKEN' && (!selectedSecretId || !selectedSecretKey)) {
      return
    }

    const integrationData: CreateIntegrationParams = {
      name: trimmedIntegrationName,
      type: selectedType as 'semaphore',
      url: trimmedIntegrationUrl,
      authType,
      tokenSecretName: authType === 'AUTH_TYPE_TOKEN' ? selectedSecretId : undefined,
      tokenSecretKey: authType === 'AUTH_TYPE_TOKEN' ? selectedSecretKey : undefined,
    }

    try {
      await createIntegrationMutation.mutateAsync(integrationData)
      showSuccessToast('Integration created successfully')
      resetForm()
      setSection('list')
    } catch (error) {
      console.error('Failed to create integration:', error)
      showErrorToast('Failed to create integration. Please try again.')
    }
  }

  const handleConfigureIntegration = (integration: IntegrationsIntegration) => {
    setEditingIntegration(integration)
    setSelectedType(integration.spec?.type || 'semaphore')
    setIntegrationName(integration.metadata?.name || '')
    setIntegrationUrl(integration.spec?.url || '')
    setAuthType(integration.spec?.auth?.use || 'AUTH_TYPE_NONE' as 'AUTH_TYPE_TOKEN' | 'AUTH_TYPE_OIDC' | 'AUTH_TYPE_NONE')
    setSelectedSecretId(integration.spec?.auth?.token?.valueFrom?.secret?.name || '')
    setSelectedSecretKey(integration.spec?.auth?.token?.valueFrom?.secret?.key || '')
    setSection('edit')
  }

  const handleUpdateIntegration = async () => {
    if (!integrationName.trim() || !integrationUrl.trim() || !editingIntegration) {
      return
    }

    // Validate token authentication requirements
    if (authType === 'AUTH_TYPE_TOKEN' && (!selectedSecretId || !selectedSecretKey)) {
      return
    }

    const updateData: UpdateIntegrationParams = {
      id: editingIntegration.metadata?.id as string,
      name: integrationName.trim(),
      type: selectedType,
      url: integrationUrl.trim(),
      authType,
      tokenSecretName: authType === 'AUTH_TYPE_TOKEN' ? selectedSecretId : undefined,
      tokenSecretKey: authType === 'AUTH_TYPE_TOKEN' ? selectedSecretKey : undefined,
    }

    try {
      await updateIntegrationMutation.mutateAsync(updateData)
      showSuccessToast('Integration updated successfully')
      resetForm()
      setSection('list')
    } catch (error) {
      console.error('Failed to update integration:', error)
      const errorMessage = (error as Error).message
        ? (error as Error).message
        : 'Failed to update integration. Please try again.'
      showErrorToast(errorMessage)
    }
  }

  const hasIntegrations = integrations && integrations.length > 0

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
        {section === 'choose-type' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">Choose integration type</span>
          </>
        )}
        {(section === 'new' || section === 'edit') && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            {section === 'new' ? (
              <button
                onClick={() => setSection('choose-type')}
                className="hover:text-zinc-900 dark:hover:text-zinc-100"
              >
                Choose integration type
              </button>
            ) : (
              <button
                onClick={() => setSection('list')}
                className="hover:text-zinc-900 dark:hover:text-zinc-100"
              >
                Integrations
              </button>
            )}
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">
              {section === 'new'
                ? `New ${INTEGRATION_TYPES[selectedType]?.label} integration`
                : `Edit ${INTEGRATION_TYPES[selectedType]?.label} integration`
              }
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
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {integrations.map((integration) => (
                <div
                  key={integration.metadata?.id}
                  className="flex flex-col justify-between bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700"
                >
                  <div>

                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-gray-200 rounded flex items-center justify-center">
                          <img className="w-8 h-8 p-2 object-contain" src={INTEGRATION_TYPES[integration.spec?.type || '']?.icon as string} alt={integration.metadata?.name} />
                        </div>
                        <Heading level={3} className="max-w-50 truncate">
                          {integration.metadata?.name}
                        </Heading>
                      </div>

                    </div>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-4 text-left">
                      {INTEGRATION_TYPES[integration.spec?.type || '']?.description}
                    </Text>
                  </div>
                  <div className="flex space-x-2 items-center">
                    <Button className="flex items-center gap-2" plain onClick={() => handleConfigureIntegration(integration)}>
                      <MaterialSymbol name="settings" />
                      Configure
                    </Button>
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
              {Object.values(INTEGRATION_TYPES).map((integrationType) => (
                <button
                  key={integrationType.value}
                  onClick={() => handleTypeSelection(integrationType.value)}
                  className="relative flex items-start gap-4 p-6 border border-zinc-200 dark:border-zinc-700 rounded-lg hover:border-blue-300 dark:hover:border-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors text-left group"
                >
                  <div className={`p-3 rounded-lg ${integrationType.color} flex items-center justify-center`}>
                    <img className="w-8 h-8 object-contain" src={integrationType.icon as string} alt={integrationType.label} />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <Heading level={3} className="text-lg group-hover:text-blue-600 dark:group-hover:text-blue-400">
                        {integrationType.label}
                      </Heading>

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

      {/* New/Edit Integration Section */}
      {(section === 'new' || section === 'edit') && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>
              {section === 'new'
                ? `New ${INTEGRATION_TYPES[selectedType]?.label} integration`
                : `Edit ${INTEGRATION_TYPES[selectedType]?.label} integration`
              }
            </Heading>
          </div>

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 text-left">
            <div className="space-y-6">
              {/* Integration Name */}
              <div>
                <label htmlFor="integration-name" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  Integration Name
                </label>
                <Input
                  id="integration-name"
                  type="text"
                  placeholder={`Enter a name for this ${INTEGRATION_TYPES[selectedType]?.label} integration`}
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
                    selectedType === 'semaphore'
                      ? 'https://api.semaphoreci.com'
                      : ''
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

              {/* Secret Selection (if token auth) */}
              {authType === 'AUTH_TYPE_TOKEN' && (
                <div className="space-y-4">
                  <div>
                    <label htmlFor="secret-select" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                      Select Secret
                    </label>
                    <select
                      id="secret-select"
                      value={selectedSecretId}
                      onChange={(e) => {
                        setSelectedSecretId(e.target.value)
                        setSelectedSecretKey('') // Reset key selection when secret changes
                      }}
                      className="mt-2 block w-full rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    >
                      <option value="">Choose a secret...</option>
                      {secrets.map((secret) => (
                        <option key={secret.metadata?.id} value={secret.metadata?.name}>
                          {secret.metadata?.name}
                        </option>
                      ))}
                    </select>
                    {secrets.length === 0 && (
                      <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                        No secrets available. Create a secret first in the &nbsp;
                        <span className="text-blue-600 hover:underline cursor-pointer" onClick={() => navigate(`/${organizationId}/canvas/${canvasId}#secrets`)}>secrets section</span>.
                      </Text>
                    )}
                  </div>

                  {/* Key Selection (if secret is selected) */}
                  {selectedSecretId && selectedSecret && (
                    <div>
                      <label htmlFor="key-select" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                        Select Key
                      </label>
                      <select
                        id="key-select"
                        value={selectedSecretKey}
                        onChange={(e) => setSelectedSecretKey(e.target.value)}
                        className="mt-2 block w-full rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 px-3 py-2 text-sm text-zinc-900 dark:text-zinc-100 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                      >
                        <option value="">Choose a key...</option>
                        {selectedSecret.spec?.local?.data && Object.keys(selectedSecret.spec.local.data).map((key) => (
                          <option key={key} value={key}>
                            {key}
                          </option>
                        ))}
                      </select>
                      <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                        Select which key from the secret to use as the authentication token.
                      </Text>
                    </div>
                  )}
                </div>
              )}

              {/* Action buttons */}
              <div className="flex gap-3">
                <Button onClick={() => section === 'edit' ? setSection('list') : setSection('choose-type')}>
                  <MaterialSymbol name="arrow_back" size="sm" />
                  Back
                </Button>
                <Button
                  color="blue"
                  onClick={section === 'edit' ? handleUpdateIntegration : handleCreateIntegration}
                  disabled={
                    (section === 'edit' ? updateIntegrationMutation.isPending : createIntegrationMutation.isPending) ||
                    !integrationName.trim() ||
                    !integrationUrl.trim() ||
                    (authType === 'AUTH_TYPE_TOKEN' && (!selectedSecretId || !selectedSecretKey))
                  }
                >
                  {(section === 'edit' ? updateIntegrationMutation.isPending : createIntegrationMutation.isPending) ? (
                    <>
                      <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
                      {section === 'edit' ? 'Updating...' : 'Creating...'}
                    </>
                  ) : (
                    <>
                      <MaterialSymbol name={section === 'edit' ? 'save' : 'add'} size="sm" />
                      {section === 'edit' ? 'Update Integration' : 'Create Integration'}
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