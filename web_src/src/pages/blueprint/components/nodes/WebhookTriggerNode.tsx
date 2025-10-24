import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'
import { getColorClass } from '../../../../utils/colors'

type WebhookTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
  icon?: string
  color?: string
}>

export const WebhookTriggerNode = memo(({ data }: NodeProps<WebhookTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]
  const icon = (data.icon as string) || 'webhook'
  const color = (data.color as string) || 'purple'

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

WebhookTriggerNode.displayName = 'WebhookTriggerNode'
