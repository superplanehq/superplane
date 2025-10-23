import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'

type GithubTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
  metadata?: Record<string, any>
}>

export const GithubTriggerNode = memo(({ data }: NodeProps<GithubTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]
  const config = data.configuration as Record<string, any> | undefined
  const metadata = data.metadata as Record<string, any> | undefined

  // Extract configuration
  const events = config?.events as string[] | undefined
  const repository = metadata?.repository as Record<string, any> | undefined

  // Format the events description
  const formatEventsDescription = () => {
    if (!events || events.length === 0) return 'No events configured'

    const eventLabels = events.map(event => {
      switch (event) {
        case 'push':
          return 'Push'
        case 'pull_request':
          return 'Pull Request'
        default:
          return event
      }
    })

    return eventLabels.join(', ')
  }

  const eventsDescription = formatEventsDescription()

  return (
    <div className="relative bg-white dark:bg-zinc-800 border-2 border-purple-400 dark:border-purple-500 rounded-lg shadow-md min-w-[180px]">
      {/* Node header */}
      <div className="px-4 py-3 bg-purple-50 dark:bg-purple-900/20">
        <div className="flex items-center gap-2">
          <MaterialSymbol name="code" size="sm" className="text-purple-600 dark:text-purple-400" />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
      </div>

      {/* Configuration */}
      <div className="px-4 py-2 border-t border-purple-200 dark:border-purple-700/50 text-xs text-purple-600 dark:text-purple-400 text-left space-y-1">
        {repository && repository.url ? (
          <a
            href={repository.url as string}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium truncate block hover:underline hover:text-purple-800 dark:hover:text-purple-200"
            title={repository.name as string}
            onClick={(e) => e.stopPropagation()}
          >
            {repository.name as string}
          </a>
        ) : repository ? (
          <div className="font-medium truncate" title={repository.name as string}>
            {repository.name as string}
          </div>
        ) : null}
        <div className="text-purple-700 dark:text-purple-300">
          {eventsDescription}
        </div>
      </div>

      {/* Output handle - centered on right edge */}
      <Handle
        type="source"
        position={Position.Right}
        id={channel}
        className="!w-3 !h-3 !bg-purple-500 !border-2 !border-white dark:!border-zinc-800"
      />
    </div>
  )
})

GithubTriggerNode.displayName = 'GithubTriggerNode'
