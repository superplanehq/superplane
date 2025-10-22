import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'

type ManualTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
}>

export const ManualTriggerNode = memo(({ data }: NodeProps<ManualTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]

  return (
    <div className="relative bg-white dark:bg-zinc-800 border-2 border-purple-400 dark:border-purple-500 rounded-lg shadow-md min-w-[180px]">
      {/* Node header */}
      <div className="px-4 py-3 bg-purple-50 dark:bg-purple-900/20">
        <div className="flex items-center gap-2">
          <MaterialSymbol name="play_circle" size="sm" className="text-purple-600 dark:text-purple-400" />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
      </div>

      {/* Info text */}
      <div className="px-4 py-2 border-t border-purple-200 dark:border-purple-700/50 text-xs text-purple-600 dark:text-purple-400 text-left">
        Manual trigger - click to emit
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

ManualTriggerNode.displayName = 'ManualTriggerNode'
