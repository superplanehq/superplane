import { memo } from 'react'
import { Handle, Position, NodeProps } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'

interface HttpNodeData {
  label: string
  component: string
  branches?: string[]
  configuration?: Record<string, any>
  onAddNode?: (sourceId: string, branch: string) => void
}

export const HttpNode = memo(({ data }: NodeProps<HttpNodeData>) => {
  const branches = data.branches || ['default']
  const branch = branches[0]
  const url = data.configuration?.url
  const method = data.configuration?.method

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
          <MaterialSymbol name="http" size="sm" className="text-blue-600 dark:text-blue-400" />
          <div className="font-semibold text-sm text-zinc-900 dark:text-zinc-100">
            {data.label}
          </div>
        </div>
      </div>

      {/* Configuration */}
      {(method || url) && (
        <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-700 space-y-1 text-xs text-slate-600 dark:text-slate-400 text-left">
          {method && (
            <div className="font-mono font-semibold uppercase">{method}</div>
          )}
          {url && (
            <div className="font-mono truncate" title={url}>{url}</div>
          )}
        </div>
      )}

      {/* Output branch */}
      <div className="border-t border-zinc-200 dark:border-zinc-700">
        <div className="relative flex items-center justify-between px-4 py-2 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors">
          <span className="text-xs font-medium text-zinc-600 dark:text-zinc-400">
            {branch}
          </span>

          {/* Output handle */}
          <Handle
            type="source"
            position={Position.Right}
            id={branch}
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

HttpNode.displayName = 'HttpNode'
