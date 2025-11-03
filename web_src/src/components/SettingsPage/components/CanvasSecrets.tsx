import { useState, useEffect } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import { useSecrets, useCreateSecret, useDeleteSecret, useUpdateSecret, useSecret } from '@/hooks/useSecrets'
import { formatRelativeTime } from '@/utils/timezone'

interface CanvasSecretsProps {
  canvasId: string
  organizationId: string
}

export function CanvasSecrets({ canvasId }: CanvasSecretsProps) {
  const [secretsSection, setSecretsSection] = useState<'list' | 'new' | 'edit'>('list')
  const [secretName, setSecretName] = useState('')
  const [editingSecretId, setEditingSecretId] = useState<string | null>(null)
  const [environmentVariables, setEnvironmentVariables] = useState<Array<{ id: string, name: string, value: string }>>([
    { id: '1', name: '', value: '' }
  ])

  // React Query hooks
  const { data: secrets = [], isLoading: loadingSecrets, error: secretsError } = useSecrets(canvasId, "DOMAIN_TYPE_CANVAS")
  const createSecretMutation = useCreateSecret(canvasId, "DOMAIN_TYPE_CANVAS")
  const deleteSecretMutation = useDeleteSecret(canvasId, "DOMAIN_TYPE_CANVAS")
  const updateSecretMutation = useUpdateSecret(canvasId, "DOMAIN_TYPE_CANVAS", editingSecretId || '')
  const { data: editingSecret, isLoading: loadingSecret } = useSecret(canvasId, "DOMAIN_TYPE_CANVAS", editingSecretId || '',)

  const isCreating = createSecretMutation.isPending
  const isDeleting = deleteSecretMutation.isPending
  const isUpdating = updateSecretMutation.isPending

  const handleCreateSecret = () => {
    // Reset form for new secret
    setSecretName('')
    setEnvironmentVariables([{ id: '1', name: '', value: '' }])
    setEditingSecretId(null)
    setSecretsSection('new')
  }

  const handleEditSecret = (secretId: string) => {
    setEditingSecretId(secretId)
    setSecretsSection('edit')
  }

  // Effect to populate form when editing
  useEffect(() => {
    if (editingSecret && secretsSection === 'edit') {
      setSecretName(editingSecret.metadata?.name || '')

      // Convert secret data to environment variables format
      const secretData = editingSecret.spec?.local?.data || {}
      const envVars = Object.entries(secretData).map(([key, value], index) => ({
        id: (index + 1).toString(),
        name: key,
        value: value as string
      }))

      // Ensure at least one empty row if no variables
      if (envVars.length === 0) {
        envVars.push({ id: '1', name: '', value: '' })
      }

      setEnvironmentVariables(envVars)
    }
  }, [editingSecret, secretsSection])

  const handleAddEnvironmentVariable = () => {
    setEnvironmentVariables(prev => [...prev, { id: Date.now().toString(), name: '', value: '' }])
  }

  const handleRemoveEnvironmentVariable = (id: string) => {
    setEnvironmentVariables(prev => prev.filter(env => env.id !== id))
  }

  const handleSaveSecret = async () => {
    try {
      const validEnvironmentVariables = environmentVariables.filter(env => env.name.trim() && env.value.trim())
      const secretData = {
        name: secretName,
        environmentVariables: validEnvironmentVariables
          .filter(env => env.name.trim() && env.value.trim() != '***')
          .map(env => ({
            name: env.name,
            value: env.value
          }))
      }

      if (secretsSection === 'edit' && editingSecretId) {
        // Update existing secret
        await updateSecretMutation.mutateAsync(secretData)
      } else {
        // Create new secret
        await createSecretMutation.mutateAsync(secretData)
      }

      // Reset form and return to list
      setSecretName('')
      setEnvironmentVariables([{ id: '1', name: '', value: '' }])
      setEditingSecretId(null)
      setSecretsSection('list')
    } catch (error) {
      console.error(`Failed to ${secretsSection === 'edit' ? 'update' : 'create'} secret:`, error)
    }
  }

  const handleDeleteSecret = async (secretId: string) => {
    try {
      if (confirm(`Are you sure you want to delete secret ${secretName}? This action cannot be undone.`)) {
        await deleteSecretMutation.mutateAsync(secretId)
      }
    } catch (error) {
      console.error('Failed to delete secret:', error)
    }
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
        {secretsSection === 'new' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">New secret</span>
          </>
        )}
        {secretsSection === 'edit' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">Edit secret</span>
          </>
        )}
      </div>

      {secretsSection === 'list' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={3} className="sm:text-lg">Secrets</Heading>
            <Button className="items-center" color="blue" onClick={handleCreateSecret} disabled={isCreating}>
              <MaterialSymbol name="add" />
              Add Secret
            </Button>
          </div>

          {secretsError && (
            <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              <Text>{secretsError instanceof Error ? secretsError.message : 'Failed to fetch secrets'}</Text>
            </div>
          )}

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 text-left">
            <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
              Manage environment variables and secrets for your canvas workflows. These values are encrypted and can be used in your stage configurations.
            </Text>

            {loadingSecrets ? (
              <div className="flex justify-center items-center h-32">
                <Text className="text-zinc-500 dark:text-zinc-400">Loading secrets...</Text>
              </div>
            ) : secrets.length === 0 ? (
              <div className="text-center py-12">
                <MaterialSymbol name="key" className="mx-auto text-zinc-400 mb-4" size="xl" />
                <Heading level={3} className="text-lg mb-2">No secrets configured</Heading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Create your first secret to store environment variables and sensitive configuration for this canvas.
                </Text>
              </div>
            ) : (
              <div className="space-y-4">
                {secrets.map((secret, index) => (
                  <div
                    key={secret.metadata?.id}
                    className={`flex items-center justify-between py-3 ${index < secrets.length - 1 ? 'border-b border-zinc-200 dark:border-zinc-700' : ''
                      }`}
                  >
                    <div>
                      <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
                        <MaterialSymbol name="key" size="sm" />
                        {secret.metadata?.name || 'Unnamed Secret'}
                      </div>
                      <div className="text-xs text-zinc-500 dark:text-zinc-400">
                        Added {formatRelativeTime(secret.metadata?.createdAt)}
                      </div>
                    </div>
                    <div className="flex space-x-2">
                      <Button plain onClick={() => handleEditSecret(secret.metadata?.id || '')}>
                        <MaterialSymbol name="edit" />
                      </Button>
                      <Button
                        plain
                        onClick={() => handleDeleteSecret(secret.metadata?.id || '')}
                        disabled={isDeleting}
                      >
                        <MaterialSymbol name="delete" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}

      {(secretsSection === 'new' || secretsSection === 'edit') && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>
              {secretsSection === 'edit' ? 'Edit secret' : 'New secret'}
            </Heading>
          </div>

          {(createSecretMutation.error || updateSecretMutation.error || deleteSecretMutation.error) && (
            <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              <Text>
                {createSecretMutation.error?.message || updateSecretMutation.error?.message || deleteSecretMutation.error?.message || 'An error occurred'}
              </Text>
            </div>
          )}

          {loadingSecret && secretsSection === 'edit' && (
            <div className="bg-blue-100 border border-blue-400 text-blue-700 px-4 py-3 rounded">
              <Text>Loading secret details...</Text>
            </div>
          )}

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 text-left">
            <div className="space-y-6">
              <div>
                <label htmlFor="secretName" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  Name of the Secret
                </label>
                <Input
                  id="secretName"
                  type="text"
                  placeholder="Enter secret name..."
                  value={secretName}
                  onChange={(e) => setSecretName(e.target.value)}
                  className="w-full"
                />
              </div>

              <div>
                <Heading level={3} className="mb-4">Environment Variables</Heading>
                <div className="space-y-3">
                  {environmentVariables.map((env, index) => (
                    <div key={env.id} className="flex gap-3 items-center">
                      <div className="flex-1">
                        <Input
                          type="text"
                          placeholder="Variable name (e.g., DATABASE_URL)"
                          value={env.name}
                          onChange={(e) => {
                            const newEnvs = [...environmentVariables]
                            newEnvs[index].name = e.target.value
                            setEnvironmentVariables(newEnvs)
                          }}
                          className="w-full"
                        />
                      </div>
                      <div className="flex-1">
                        <Input
                          type="password"
                          placeholder="Variable value"
                          value={env.value}
                          onChange={(e) => {
                            const newEnvs = [...environmentVariables]
                            newEnvs[index].value = e.target.value
                            setEnvironmentVariables(newEnvs)
                          }}
                          className="w-full"
                        />
                      </div>
                      {environmentVariables.length > 1 && (
                        <Button
                          plain
                          onClick={() => handleRemoveEnvironmentVariable(env.id)}
                          className="text-red-500 hover:text-red-700"
                        >
                          <MaterialSymbol name="delete" size="sm" />
                        </Button>
                      )}
                    </div>
                  ))}
                  <Button plain onClick={handleAddEnvironmentVariable} className="text-blue-600 hover:text-blue-800">
                    <MaterialSymbol name="add" size="sm" />
                    Add Variable
                  </Button>
                </div>
                <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-3">
                  These variables will be available in your canvas stages and can be referenced using environment variable syntax.
                </Text>
              </div>

              <div className="flex gap-3">
                <Button
                  onClick={() => {
                    setSecretsSection('list')
                    setEditingSecretId(null)
                  }}
                  disabled={isCreating || isUpdating}
                >
                  Cancel
                </Button>
                <Button
                  color="blue"
                  onClick={handleSaveSecret}
                  disabled={!secretName.trim() || isCreating || isUpdating || (secretsSection === 'edit' && loadingSecret)}
                >
                  {secretsSection === 'edit'
                    ? (isUpdating ? 'Updating...' : 'Update Secret')
                    : (isCreating ? 'Creating...' : 'Save Secret')
                  }
                </Button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}