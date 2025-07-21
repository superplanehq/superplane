import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'

interface CanvasIntegrationsProps {
  canvasId: string
  organizationId: string
}

export function CanvasIntegrations({ canvasId, organizationId }: CanvasIntegrationsProps) {
  const handleAddIntegration = () => {
    // TODO: Implement integration management
    console.log('Adding integration for canvas:', canvasId, 'in organization:', organizationId)
  }

  return (
    <div className="space-y-6">
      <Heading level={2}>Integrations</Heading>
      
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
        <div className="text-center py-12">
          <MaterialSymbol name="integration_instructions" className="mx-auto text-zinc-400 mb-4" size="xl" />
          <Heading level={3} className="text-lg mb-2">No integrations configured</Heading>
          <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
            Connect external services like webhooks, databases, APIs, and third-party tools to enhance your canvas workflows.
          </Text>
          <Button color="blue" onClick={handleAddIntegration}>
            <MaterialSymbol name="add" size="sm" />
            Add Integration
          </Button>
        </div>
      </div>

      {/* Future: Integration categories */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-4 opacity-60">
          <div className="flex items-center gap-3 mb-3">
            <MaterialSymbol name="webhook" className="text-blue-500" size="lg" />
            <div>
              <Heading level={4} className="text-sm">Webhooks</Heading>
              <Text className="text-xs text-zinc-500">HTTP callbacks</Text>
            </div>
          </div>
          <Text className="text-xs text-zinc-600 dark:text-zinc-400">
            Coming soon - Configure HTTP endpoints to receive real-time notifications from your workflows.
          </Text>
        </div>

        <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-4 opacity-60">
          <div className="flex items-center gap-3 mb-3">
            <MaterialSymbol name="storage" className="text-green-500" size="lg" />
            <div>
              <Heading level={4} className="text-sm">Databases</Heading>
              <Text className="text-xs text-zinc-500">Data persistence</Text>
            </div>
          </div>
          <Text className="text-xs text-zinc-600 dark:text-zinc-400">
            Coming soon - Connect to PostgreSQL, MySQL, MongoDB and other database systems.
          </Text>
        </div>

        <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-4 opacity-60">
          <div className="flex items-center gap-3 mb-3">
            <MaterialSymbol name="api" className="text-purple-500" size="lg" />
            <div>
              <Heading level={4} className="text-sm">APIs</Heading>
              <Text className="text-xs text-zinc-500">External services</Text>
            </div>
          </div>
          <Text className="text-xs text-zinc-600 dark:text-zinc-400">
            Coming soon - Integrate with REST APIs, GraphQL endpoints, and third-party services.
          </Text>
        </div>
      </div>
    </div>
  )
}