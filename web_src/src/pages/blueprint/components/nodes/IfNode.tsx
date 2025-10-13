import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'

type IfNodeData = Node<{
  label?: string
  component?: string
  channels?: string[]
  configuration?: Record<string, any>
}>

export const IfNode = memo(({ data }: NodeProps<IfNodeData>) => {
  const channels = (data.channels as string[]) || ['true', 'false']
  const expression = (data.configuration as Record<string, any>)?.expression

  return (
    <div className="bg-white dark:bg-zinc-800 border-2 border-zinc-400 dark:border-zinc-500 rounded-lg shadow-md min-w-[180px]">
      {/* Input handle */}
      <Handle
        type="target"
        position={Position.Left}
        className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800"
      />

      {/* Node header */}
      <div className="px-4 py-3 bg-zinc-50 dark:bg-zinc-900/20">
        <div className="flex items-center gap-2">
          <MaterialSymbol name="alt_route" size="sm" className="text-blue-600 dark:text-blue-400" />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label as string}
          </div>
        </div>
      </div>

      {/* Configuration */}
      {expression && (
        <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-slate-600 dark:text-slate-400 font-mono truncate text-left" title={expression}>
          {expression}
        </div>
      )}

      {/* Output channels */}
      <div className="border-t border-zinc-200 dark:border-zinc-700">
        {channels.map((channel, index) => (
          <div
            key={channel}
            className="relative flex items-center justify-between px-4 py-2 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors"
            style={{
              borderTop: index > 0 ? '1px solid' : 'none',
              borderColor: 'var(--zinc-200)',
            }}
          >
            <span className="text-xs font-medium text-zinc-600 dark:text-zinc-400">
              {channel}
            </span>

            {/* Output handle */}
            <Handle
              type="source"
              position={Position.Right}
              id={channel}
              className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800 !right-[-6px]"
              style={{
                top: '50%',
                transform: 'translateY(-50%)',
              }}
            />
          </div>
        ))}
      </div>
    </div>
  )
})

IfNode.displayName = 'IfNode'
