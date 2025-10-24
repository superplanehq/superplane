import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'
import { getColorClass } from '../../../../utils/colors'

type ApprovalNodeData = Node<{
  label?: string
  configuration?: Record<string, any>
  icon?: string
  color?: string
}>

export const ApprovalNode = memo(({ data }: NodeProps<ApprovalNodeData>) => {
  const icon = (data.icon as string) || 'check'
  const color = (data.color as string) || 'blue'

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
          <MaterialSymbol name={icon} size="sm" className={getColorClass(color)} />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label}
          </div>
        </div>
      </div>

      {/* Configuration */}
      {data.configuration?.count && (
        <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-slate-600 dark:text-slate-400 text-left">
          Requires {data.configuration.count} approval{data.configuration.count !== 1 ? 's' : ''}
        </div>
      )}

      {/* Output channel */}
      <div className="border-t border-zinc-200 dark:border-zinc-700">
        <div className="relative flex items-center justify-between px-4 py-2 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors">
          <span className="text-xs font-medium text-zinc-600 dark:text-zinc-400">default</span>

          {/* Output handle */}
          <Handle
            type="source"
            position={Position.Right}
            id="default"
            className="!w-3 !h-3 !bg-slate-500 !border-2 !border-white dark:!border-zinc-800 !right-[-6px]"
            style={{
              top: '50%',
              transform: 'translateY(-50%)',
            }}
          />
        </div>
      </div>
    </div>
  )
})

ApprovalNode.displayName = 'ApprovalNode'
