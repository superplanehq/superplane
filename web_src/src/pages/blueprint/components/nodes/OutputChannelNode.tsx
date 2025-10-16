import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'

type OutputChannelNodeData = Node<{
  label?: string
  description?: string
}>

export const OutputChannelNode = memo(({ data }: NodeProps<OutputChannelNodeData>) => {
  return (
    <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 border-2 border-green-500 dark:border-green-600 rounded-lg shadow-md min-w-[180px]">
      {/* Input handle only - no output handle for output channels */}
      <Handle
        type="target"
        position={Position.Left}
        className="!w-3 !h-3 !bg-green-500 !border-2 !border-white dark:!border-zinc-800"
      />

      {/* Node header */}
      <div className="px-4 py-3">
        <div className="flex items-center gap-2 mb-1">
          <MaterialSymbol name="output" size="sm" className="text-green-600 dark:text-green-400" />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
        {data.description && (
          <div className="text-xs text-zinc-600 dark:text-zinc-400 mt-1">
            {data.description as string}
          </div>
        )}
        <div className="text-xs text-green-600 dark:text-green-400 mt-2 font-medium">
          Output Channel
        </div>
      </div>
    </div>
  )
})

OutputChannelNode.displayName = 'OutputChannelNode'
