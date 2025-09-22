import { useState, useRef } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { useIntegrations, useCreateIntegration, useUpdateIntegration, type UpdateIntegrationParams } from '../../../pages/canvas/hooks/useIntegrations'
import { useSecrets, useCreateSecret, useUpdateSecret } from '../../../pages/canvas/hooks/useSecrets'
import { IntegrationsIntegration } from '@/api-client'
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg'
import GithubLogo from '@/assets/github-mark.svg'

import {
  GitHubIntegrationForm,
  SemaphoreIntegrationForm,
  SettingsApiTokenForm,
  useIntegrationForm,
  NEW_SECRET_NAME
} from '../../IntegrationForm'
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
  const [editingIntegration, setEditingIntegration] = useState<IntegrationsIntegration | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [newSecretValue, setNewSecretValue] = useState('')
  const orgUrlRef = useRef<HTMLInputElement>(null)

  const { data: integrations, isLoading, error } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS")
  const createIntegrationMutation = useCreateIntegration(canvasId, "DOMAIN_TYPE_CANVAS")
  const updateIntegrationMutation = useUpdateIntegration(canvasId, "DOMAIN_TYPE_CANVAS", editingIntegration?.metadata?.id || '')
  const { data: secrets = [] } = useSecrets(canvasId, "DOMAIN_TYPE_CANVAS")
  const createSecretMutation = useCreateSecret(canvasId, "DOMAIN_TYPE_CANVAS")
  const updateSecretMutation = useUpdateSecret(canvasId, "DOMAIN_TYPE_CANVAS", editingIntegration?.spec?.auth?.token?.valueFrom?.secret?.name || '')

  const {
    integrationData,
    setIntegrationData,
    apiTokenTab,
    setApiTokenTab,
    newSecretToken,
    setNewSecretToken,
    errors,
    setErrors,
    validateForm,
    resetForm
  } = useIntegrationForm({
    integrationType: selectedType,
    integrations: integrations || [],
    editingIntegration
  })

  const handleAddIntegration = () => {
    setSection('choose-type')
  }

  const handleTypeSelection = (type: string) => {
    setSelectedType(type)
    resetForm()
    setSection('new')
  }

  const handleCreateIntegration = async () => {
    if (!validateForm()) {
      return
    }

    setIsCreating(true)

    try {
      let secretName = integrationData.apiToken.secretName
      let secretKey = integrationData.apiToken.secretKey

      if (apiTokenTab === 'new') {
        try {
          const secretData = {
            name: `${integrationData.name.trim()}-api-key`,
            environmentVariables: [{
              name: NEW_SECRET_NAME,
              value: newSecretToken
            }]
          }

          await createSecretMutation.mutateAsync(secretData)
          secretName = secretData.name
          secretKey = NEW_SECRET_NAME
        } catch {
          setErrors({ apiToken: 'Failed to create a secret, please try to create secret manually and import' })
          return
        }
      }

      let trimmedUrl = integrationData.orgUrl.trim()
      if (trimmedUrl.endsWith('/')) {
        trimmedUrl = trimmedUrl.slice(0, -1)
      }

      const integrationPayload = {
        name: integrationData.name.trim(),
        type: selectedType,
        url: trimmedUrl,
        authType: 'AUTH_TYPE_TOKEN' as const,
        tokenSecretName: secretName,
        tokenSecretKey: secretKey
      }

      await createIntegrationMutation.mutateAsync(integrationPayload)
      showSuccessToast('Integration created successfully')
      resetForm()
      setSection('list')
    } catch (error) {
      console.error('Failed to create integration:', error)
      showErrorToast('Failed to create integration. Please try again.')
    } finally {
      setIsCreating(false)
    }
  }

  const handleConfigureIntegration = (integration: IntegrationsIntegration) => {
    setEditingIntegration(integration)
    setSelectedType(integration.spec?.type || 'semaphore')

    setIntegrationData({
      orgUrl: integration.spec?.url || '',
      name: integration.metadata?.name || '',
      apiToken: {
        secretName: integration.spec?.auth?.token?.valueFrom?.secret?.name || '',
        secretKey: integration.spec?.auth?.token?.valueFrom?.secret?.key || ''
      }
    })

    setApiTokenTab('existing')
    setNewSecretValue('')

    setSection('edit')
  }

  const handleUpdateIntegration = async () => {
    if (!validateForm() || !editingIntegration) {
      return
    }

    setIsCreating(true)

    try {
      // Check if only the secret value has changed
      const currentUrl = editingIntegration.spec?.url || ''
      const currentName = editingIntegration.metadata?.name || ''
      let trimmedUrl = integrationData.orgUrl.trim()
      if (trimmedUrl.endsWith('/')) {
        trimmedUrl = trimmedUrl.slice(0, -1)
      }

      const isOnlySecretUpdate = newSecretValue.trim() &&
        integrationData.apiToken.secretName &&
        integrationData.apiToken.secretKey &&
        currentUrl === trimmedUrl &&
        currentName === integrationData.name.trim()

      if (isOnlySecretUpdate) {
        // Only update the secret
        try {
          await updateSecretMutation.mutateAsync({
            name: integrationData.apiToken.secretName,
            environmentVariables: [{
              name: integrationData.apiToken.secretKey,
              value: newSecretValue.trim()
            }]
          });
          showSuccessToast('Secret updated successfully')
        } catch (error) {
          console.error('Failed to update secret:', error);
          setErrors({ secretValue: 'Failed to update secret value' });
          return;
        }
      } else {
        // Update secret if provided
        if (newSecretValue.trim() && integrationData.apiToken.secretName && integrationData.apiToken.secretKey) {
          try {
            await updateSecretMutation.mutateAsync({
              name: integrationData.apiToken.secretName,
              environmentVariables: [{
                name: integrationData.apiToken.secretKey,
                value: newSecretValue.trim()
              }]
            });
          } catch (error) {
            console.error('Failed to update secret:', error);
            setErrors({ secretValue: 'Failed to update secret value' });
            return;
          }
        }
        const updateData: UpdateIntegrationParams = {
          id: editingIntegration.metadata?.id as string,
          name: integrationData.name.trim(),
          type: selectedType,
          url: trimmedUrl,
          authType: 'AUTH_TYPE_TOKEN' as const,
          tokenSecretName: integrationData.apiToken.secretName,
          tokenSecretKey: integrationData.apiToken.secretKey,
        }

        await updateIntegrationMutation.mutateAsync(updateData)
        showSuccessToast('Integration updated successfully')
      }

      resetForm()
      setEditingIntegration(null)
      setNewSecretValue('')
      setSection('list')
    } catch (error) {
      console.error('Failed to update integration:', error)
      const errorMessage = (error as Error).message
        ? (error as Error).message
        : 'Failed to update integration. Please try again.'
      showErrorToast(errorMessage)
    } finally {
      setIsCreating(false)
    }
  }

  const hasIntegrations = integrations && integrations.length > 0

  return (
    <div className="max-w-4xl mx-auto space-y-6 h-full overflow-y-auto px-4 pb-8">
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
              <Button onClick={() => {
                resetForm()
                setSection('list')
              }}>
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
              {/* Integration Type Specific Form */}
              {selectedType === 'github' ? (
                <GitHubIntegrationForm
                  integrationData={integrationData}
                  setIntegrationData={setIntegrationData}
                  errors={errors}
                  setErrors={setErrors}
                  apiTokenTab={apiTokenTab}
                  setApiTokenTab={setApiTokenTab}
                  newSecretToken={newSecretToken}
                  setNewSecretToken={setNewSecretToken}
                  secrets={secrets}
                  orgUrlRef={orgUrlRef}
                />
              ) : (
                <SemaphoreIntegrationForm
                  integrationData={integrationData}
                  setIntegrationData={setIntegrationData}
                  errors={errors}
                  setErrors={setErrors}
                  apiTokenTab={apiTokenTab}
                  setApiTokenTab={setApiTokenTab}
                  newSecretToken={newSecretToken}
                  setNewSecretToken={setNewSecretToken}
                  secrets={secrets}
                  orgUrlRef={orgUrlRef}
                />
              )}

              {/* API Token Form */}
              <SettingsApiTokenForm
                integrationData={integrationData}
                setIntegrationData={setIntegrationData}
                errors={errors}
                setErrors={setErrors}
                secrets={secrets}
                organizationId={organizationId}
                canvasId={canvasId}
                isEditMode={section === 'edit'}
                newSecretValue={newSecretValue}
                setNewSecretValue={setNewSecretValue}
              />

              {/* Action buttons */}
              <div className="flex gap-3">
                <Button onClick={() => {
                  if (section === 'edit') {
                    resetForm()
                    setEditingIntegration(null)
                    setSection('list')
                  } else {
                    resetForm()
                    setSection('choose-type')
                  }
                }}>
                  <MaterialSymbol name="arrow_back" size="sm" />
                  Back
                </Button>
                <Button
                  color="blue"
                  onClick={section === 'edit' ? handleUpdateIntegration : handleCreateIntegration}
                  disabled={
                    isCreating ||
                    (section === 'edit' ? updateIntegrationMutation.isPending : createIntegrationMutation.isPending) ||
                    !integrationData.name.trim() ||
                    !integrationData.orgUrl.trim() ||
                    (apiTokenTab === 'existing' && (!integrationData.apiToken.secretName || !integrationData.apiToken.secretKey)) ||
                    (apiTokenTab === 'new' && !newSecretToken.trim())
                  }
                >
                  {(section === 'edit' ? updateIntegrationMutation.isPending : createIntegrationMutation.isPending) || isCreating ? (
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