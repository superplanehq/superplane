import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'
import { getColorClass } from '../../../../utils/colors'

type SemaphoreTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
  metadata?: Record<string, any>
  icon?: string
  color?: string
}>

export const SemaphoreTriggerNode = memo(({ data }: NodeProps<SemaphoreTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]
  const metadata = data.metadata as Record<string, any> | undefined
  const icon = (data.icon as string) || 'deployed_code'
  const color = (data.color as string) || 'purple'

  // Extract metadata
  const project = metadata?.project as Record<string, any> | undefined

  return (
    <div className="relative bg-white dark:bg-zinc-800 border-2 border-zinc-400 dark:border-zinc-500 rounded-lg shadow-md min-w-[180px]">
      {/* Node header */}
      <div className="px-4 py-3 bg-zinc-50 dark:bg-zinc-900/20">
        <div className="flex items-center gap-2">
          <MaterialSymbol name={icon} size="sm" className={getColorClass(color)} />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
      </div>

      {/* Configuration */}
      <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-zinc-600 dark:text-zinc-400 text-left space-y-1">
        {project && project.url ? (
          <a
            href={project.url as string}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium truncate block hover:underline hover:text-zinc-800 dark:hover:text-zinc-200"
            title={project.name as string}
            onClick={(e) => e.stopPropagation()}
          >
            {project.name as string}
          </a>
        ) : project ? (
          <div className="font-medium truncate" title={project.name as string}>
            {project.name as string}
          </div>
        ) : (
          <div>Semaphore trigger</div>
        )}
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

SemaphoreTriggerNode.displayName = 'SemaphoreTriggerNode'
