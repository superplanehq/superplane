import { memo } from 'react'
import { Handle, Position, NodeProps, Node } from '@xyflow/react'
import { MaterialSymbol } from '../../../../components/MaterialSymbol/material-symbol'
import { getColorClass } from '../../../../utils/colors'

type ScheduledTriggerNodeData = Node<{
  label?: string
  channels?: string[]
  configuration?: Record<string, any>
  metadata?: Record<string, any>
  icon?: string
  color?: string
}>

export const ScheduledTriggerNode = memo(({ data }: NodeProps<ScheduledTriggerNodeData>) => {
  const channels = (data.channels as string[]) || ['default']
  const channel = channels[0]
  const config = data.configuration as Record<string, any> | undefined
  const metadata = data.metadata as Record<string, any> | undefined
  const icon = (data.icon as string) || 'schedule'
  const color = (data.color as string) || 'purple'

  // Extract schedule configuration
  const scheduleType = config?.type as string | undefined
  const minute = config?.minute as number | undefined
  const time = config?.time as string | undefined
  const weekDay = config?.weekDay as string | undefined
  const nextTrigger = metadata?.nextTrigger as string | undefined

  // Format the schedule description
  const formatScheduleDescription = () => {
    if (!scheduleType) return 'Scheduled trigger'

    switch (scheduleType) {
      case 'hourly':
        return minute !== undefined ? `Hourly at :${minute.toString().padStart(2, '0')}` : 'Hourly'
      case 'daily':
        return time ? `Daily at ${time} UTC` : 'Daily'
      case 'weekly':
        const dayLabel = weekDay ? weekDay.charAt(0).toUpperCase() + weekDay.slice(1).toLowerCase() : ''
        return time && weekDay ? `${dayLabel}s at ${time} UTC` : 'Weekly'
      default:
        return 'Scheduled trigger'
    }
  }

  // Format the next trigger time
  const formatNextTrigger = (timestamp: string | undefined) => {
    if (!timestamp) return null
    try {
      const date = new Date(timestamp)
      const now = new Date()
      const diffMs = date.getTime() - now.getTime()
      const diffMins = Math.floor(diffMs / 60000)

      if (diffMins < 0) {
        return 'Triggering soon...'
      } else if (diffMins < 60) {
        return `Next: in ${diffMins}m`
      } else if (diffMins < 1440) {
        const hours = Math.floor(diffMins / 60)
        return `Next: in ${hours}h`
      } else {
        return `Next: ${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
      }
    } catch (e) {
      return null
    }
  }

  const scheduleDescription = formatScheduleDescription()
  const nextTriggerText = formatNextTrigger(nextTrigger)

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
        <div>{scheduleDescription}</div>
        {nextTriggerText && (
          <div className="font-semibold text-zinc-700 dark:text-zinc-300" title={nextTrigger}>
            {nextTriggerText}
          </div>
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

ScheduledTriggerNode.displayName = 'ScheduledTriggerNode'
