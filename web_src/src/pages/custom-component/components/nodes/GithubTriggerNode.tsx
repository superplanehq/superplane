import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { resolveIcon } from '../../../../lib/utils'
import { getColorClass } from '../../../../utils/colors'

type GithubTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
  metadata?: Record<string, any>
  icon?: string
  color?: string
}>

export const GithubTriggerNode = memo(({ data }: NodeProps<GithubTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]
  const config = data.configuration as Record<string, any> | undefined
  const metadata = data.metadata as Record<string, any> | undefined
  const icon = (data.icon as string) || 'github'
  const color = (data.color as string) || 'purple'
  const IconComponent = resolveIcon(icon)

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
    <div className="relative bg-white dark:bg-zinc-800 border-2 border-zinc-400 dark:border-zinc-500 rounded-lg shadow-md min-w-[180px]">
      {/* Node header */}
      <div className="px-4 py-3 bg-zinc-50 dark:bg-zinc-900/20">
        <div className="flex items-center gap-2">
          <IconComponent size={20} className={getColorClass(color)} />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
      </div>

      {/* Configuration */}
      <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-zinc-600 dark:text-zinc-400 text-left space-y-1">
        {repository && repository.url ? (
          <a
            href={repository.url as string}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium truncate block hover:underline hover:text-zinc-800 dark:hover:text-zinc-200"
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
        <div className="text-zinc-700 dark:text-zinc-300">
          {eventsDescription}
        </div>
      </div>

      {/* Output handle - centered on right edge */}
      <Handle
        type="source"
        position={Position.Right}
        id={channel}
        className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800"
      />
    </div>
  )
})

GithubTriggerNode.displayName = 'GithubTriggerNode'
