import { useState } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import { Textarea } from '../../Textarea/textarea'

interface CanvasSecretsProps {
  canvasId: string
  organizationId: string
}

export function CanvasSecrets({ canvasId, organizationId }: CanvasSecretsProps) {
  const [secretsSection, setSecretsSection] = useState<'list' | 'new'>('list')
  const [secretName, setSecretName] = useState('')
  const [secretDescription, setSecretDescription] = useState('')
  const [environmentVariables, setEnvironmentVariables] = useState<Array<{id: string, name: string, value: string}>>([
    { id: '1', name: '', value: '' }
  ])

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
    // TODO: Implement API call to save secret
    console.log('Saving secret:', { secretName, secretDescription, environmentVariables, canvasId, organizationId })
    
    // Reset form
    setSecretName('')
    setSecretDescription('')
    setEnvironmentVariables([{ id: '1', name: '', value: '' }])
    setSecretsSection('list')
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
        <button 
          className={secretsSection === 'list' ? 'font-medium text-zinc-900 dark:text-zinc-100' : 'hover:text-zinc-900 dark:hover:text-zinc-100'}
          onClick={() => setSecretsSection('list')}
        >
          Secrets
        </button>
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
            <Heading level={2}>Secrets</Heading>
            <Button color="blue" onClick={handleCreateSecret}>
              <MaterialSymbol name="add" />
              Add Secret
            </Button>
          </div>
          
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
              Manage environment variables and secrets for your canvas workflows. These values are encrypted and can be used in your stage configurations.
            </Text>
            
            <div className="space-y-4">
              <div className="text-center py-12">
                <MaterialSymbol name="key" className="mx-auto text-zinc-400 mb-4" size="xl" />
                <Heading level={3} className="text-lg mb-2">No secrets configured</Heading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Create your first secret to store environment variables and sensitive configuration for this canvas.
                </Text>
              </div>
            </div>
          </div>
        </>
      )}

      {secretsSection === 'new' && (
        <>
          <div className="flex items-center justify-between">
            <Heading level={2}>New secret</Heading>
          </div>
          
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
                <label htmlFor="secretDescription" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                  Description of the Secret
                </label>
                <Textarea
                  id="secretDescription"
                  placeholder="Describe the purpose of this secret..."
                  value={secretDescription}
                  onChange={(e) => setSecretDescription(e.target.value)}
                  className="w-full"
                  rows={3}
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
                <Button onClick={() => setSecretsSection('list')}>
                  Cancel
                </Button>
                <Button color="blue" onClick={handleSaveSecret} disabled={!secretName.trim()}>
                  Save Secret
                </Button>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}