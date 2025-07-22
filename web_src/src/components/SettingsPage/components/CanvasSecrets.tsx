import { useState } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../Table/table'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
} from '../../Dropdown/dropdown'
import { useSecrets, useCreateSecret, useDeleteSecret } from '../../../pages/canvas/hooks/useSecrets'

interface CanvasSecretsProps {
  canvasId: string
  organizationId: string
}

export function CanvasSecrets({ canvasId }: CanvasSecretsProps) {
  const [secretsSection, setSecretsSection] = useState<'list' | 'new'>('list')
  const [secretName, setSecretName] = useState('')
  const [environmentVariables, setEnvironmentVariables] = useState<Array<{ id: string, name: string, value: string }>>([
    { id: '1', name: '', value: '' }
  ])

  // React Query hooks
  const { data: secrets = [], isLoading: loadingSecrets, error: secretsError } = useSecrets(canvasId)
  const createSecretMutation = useCreateSecret(canvasId)
  const deleteSecretMutation = useDeleteSecret(canvasId)

  const isCreating = createSecretMutation.isPending
  const isDeleting = deleteSecretMutation.isPending

  const handleCreateSecret = () => {
    setSecretsSection('new')
  }

  const handleAddEnvironmentVariable = () => {
    setEnvironmentVariables(prev => [...prev, { id: Date.now().toString(), name: '', value: '' }])
  }

  const handleRemoveEnvironmentVariable = (id: string) => {
    setEnvironmentVariables(prev => prev.filter(env => env.id !== id))
  }

  const handleSaveSecret = async () => {
    try {
      const validEnvironmentVariables = environmentVariables.filter(env => env.name.trim() && env.value.trim())
      
      await createSecretMutation.mutateAsync({
        name: secretName,
        environmentVariables: validEnvironmentVariables.map(env => ({
          name: env.name,
          value: env.value
        }))
      })

      // Reset form
      setSecretName('')
      setEnvironmentVariables([{ id: '1', name: '', value: '' }])
      setSecretsSection('list')
    } catch (error) {
      console.error('Failed to create secret:', error)
    }
  }

  const handleDeleteSecret = async (secretId: string) => {
    try {
      await deleteSecretMutation.mutateAsync(secretId)
    } catch (error) {
      console.error('Failed to delete secret:', error)
    }
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
        {secretsSection === 'new' && (
          <>
            <MaterialSymbol name="chevron_right" size="sm" />
            <span className="font-medium text-zinc-900 dark:text-zinc-100">New secret</span>
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

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <Text className="text-zinc-600 text-left dark:text-zinc-400 mb-4">
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
              <Table dense>
                <TableHead>
                  <TableRow>
                    <TableHeader>Name</TableHeader>
                    <TableHeader>Description</TableHeader>
                    <TableHeader>Variables</TableHeader>
                    <TableHeader>Created</TableHeader>
                    <TableHeader></TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {secrets.map((secret) => (
                    <TableRow key={secret.metadata?.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <MaterialSymbol name="key" className="text-zinc-400" size="sm" />
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-white">
                              {secret.metadata?.name || 'Unnamed Secret'}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                          No description
                        </Text>
                      </TableCell>
                      <TableCell>
                        <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                          {Object.keys(secret.spec?.local?.data || {}).length} variables
                        </Text>
                      </TableCell>
                      <TableCell>
                        <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                          {secret.metadata?.createdAt ? new Date(secret.metadata.createdAt).toLocaleDateString() : 'Unknown'}
                        </Text>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain className="flex items-center gap-2 text-sm">
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem
                                onClick={() => handleDeleteSecret(secret.metadata?.id || '')}
                                disabled={isDeleting}
                              >
                                <MaterialSymbol name="delete" />
                                Delete
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </>
      )}

      {secretsSection === 'new' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>New secret</Heading>
          </div>

          {(createSecretMutation.error || deleteSecretMutation.error) && (
            <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              <Text>
                {createSecretMutation.error?.message || deleteSecretMutation.error?.message || 'An error occurred'}
              </Text>
            </div>
          )}

          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
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
                          type="text"
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
                <Button onClick={() => setSecretsSection('list')} disabled={isCreating}>
                  Cancel
                </Button>
                <Button 
                  color="blue" 
                  onClick={handleSaveSecret} 
                  disabled={!secretName.trim() || isCreating}
                >
                  {isCreating ? 'Creating...' : 'Save Secret'}
                </Button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}