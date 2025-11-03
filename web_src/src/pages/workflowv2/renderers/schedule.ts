import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { convertUTCToLocalTime, formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";

type ScheduleConfigurationType = "minutes" | "hourly" | "daily" | "weekly";

interface ScheduleConfiguration {
  type: ScheduleConfigurationType
  interval?: number
  minute?: number
  time?: string
  weekDay?: string
}

function formatScheduleDescription(configuration: ScheduleConfiguration): string {
  if (!configuration.type) {
    return ''
  }

  switch (configuration.type) {
    case 'minutes': {
      return configuration.interval !== undefined ? `Every ${configuration.interval} minute${configuration.interval === 1 ? '' : 's'}` : 'Every X minutes'
    }
    case 'hourly': {
      return configuration.minute !== undefined ? `Hourly at :${configuration.minute.toString().padStart(2, '0')}` : 'Hourly'
    }
    case 'daily': {
      return configuration.time ? `Daily at ${convertUTCToLocalTime(configuration.time)}` : 'Daily'
    }
    case 'weekly': {
      const dayLabel = configuration.weekDay ? configuration.weekDay.charAt(0).toUpperCase() + configuration.weekDay.slice(1).toLowerCase() : ''
      return configuration.time && configuration.weekDay ? `${dayLabel}s at ${convertUTCToLocalTime(configuration.time)}` : 'Weekly'
    }
    default:
      return 'Scheduled trigger'
  }
}

function calculateNextTrigger(configuration: ScheduleConfiguration, referenceNextTrigger?: string): Date | null {
  const now = new Date()

  switch (configuration.type) {
    case 'minutes': {
      if (configuration.interval === undefined) return null

      const interval = configuration.interval

      if (referenceNextTrigger) {
        try {
          const reference = new Date(referenceNextTrigger)
          const minutesElapsed = Math.floor((now.getTime() - reference.getTime()) / 60000)

          if (minutesElapsed < 0) {
            return reference
          }

          const completedIntervals = Math.floor(minutesElapsed / interval)
          const nextTriggerMinutes = (completedIntervals + 1) * interval
          const nextTrigger = new Date(reference.getTime() + (nextTriggerMinutes * 60000))
          return nextTrigger
        } catch {
          return null
        }
      }

      const nextMinuteRounded = new Date(now)
      nextMinuteRounded.setSeconds(0)
      nextMinuteRounded.setMilliseconds(0)
      nextMinuteRounded.setMinutes(nextMinuteRounded.getMinutes() + 1)

      const minutesSinceMidnight = nextMinuteRounded.getHours() * 60 + nextMinuteRounded.getMinutes()
      const intervalsPassed = Math.floor(minutesSinceMidnight / interval)
      const nextIntervalMinutes = (intervalsPassed + 1) * interval
      
      if (nextIntervalMinutes >= 1440) {
        const nextDay = new Date(nextMinuteRounded)
        nextDay.setDate(nextDay.getDate() + 1)
        nextDay.setHours(0)
        nextDay.setMinutes(interval)
        return nextDay
      }

      const intervalHours = Math.floor(nextIntervalMinutes / 60)
      const intervalMinutes = nextIntervalMinutes % 60
      const nextTrigger = new Date(nextMinuteRounded)
      nextTrigger.setHours(intervalHours)
      nextTrigger.setMinutes(intervalMinutes)

      return nextTrigger
    }

    case 'hourly': {
      const nextHour = new Date(now)
      nextHour.setMinutes(configuration.minute ?? 0)
      nextHour.setSeconds(0)
      nextHour.setMilliseconds(0)

      if (nextHour <= now) {
        nextHour.setHours(nextHour.getHours() + 1)
      }
      return nextHour
    }

    case 'daily': {
      if (!configuration.time) return null

      const [dailyHours, dailyMinutes] = configuration.time.split(':').map(Number)
      const nextDay = new Date(now)
      nextDay.setUTCHours(dailyHours)
      nextDay.setUTCMinutes(dailyMinutes)
      nextDay.setSeconds(0)
      nextDay.setMilliseconds(0)

      if (nextDay <= now) {
        nextDay.setDate(nextDay.getDate() + 1)
      }
      return nextDay
    }

    case 'weekly': {
      if (!configuration.time || !configuration.weekDay) return null

      const [weeklyHours, weeklyMinutes] = configuration.time.split(':').map(Number)
      const dayNames = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday']
      const targetDayIndex = dayNames.indexOf(configuration.weekDay.toLowerCase())

      if (targetDayIndex === -1) return null

      const nextWeek = new Date(now)
      const currentDayIndex = nextWeek.getDay()
      let daysUntilTarget = targetDayIndex - currentDayIndex

      if (daysUntilTarget < 0 || (daysUntilTarget === 0 && now.getUTCHours() * 60 + now.getUTCMinutes() >= weeklyHours * 60 + weeklyMinutes)) {
        daysUntilTarget += 7
      }

      nextWeek.setDate(nextWeek.getDate() + daysUntilTarget)
      nextWeek.setUTCHours(weeklyHours)
      nextWeek.setUTCMinutes(weeklyMinutes)
      nextWeek.setSeconds(0)
      nextWeek.setMilliseconds(0)

      return nextWeek
    }

    default:
      return null
  }
}

function formatNextTrigger(configuration: ScheduleConfiguration, metadata?: { nextTrigger?: string }): string {
  
  const nextTrigger = calculateNextTrigger(
    configuration,
    configuration.type === 'minutes' ? metadata?.nextTrigger : undefined
  )

  if (!nextTrigger) {
    return "-"
  }

  try {
    const now = new Date()
    const diffMs = nextTrigger.getTime() - now.getTime()
    const diffMins = Math.floor(diffMs / 60000)

    if (diffMins <= 0) {
      return 'Triggering soon...'
    }

    if (diffMins < 60) {
      return `Next: in ${diffMins}m`
    }

    if (diffMins < 1440) {
      return `Next: in ${Math.floor(diffMins / 60)}h`
    }

    return formatTimestampInUserTimezone(nextTrigger.toISOString())
  } catch {
    return ""
  }
}

/**
 * Renderer for the "schedule" trigger type
 */
export const scheduleTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    return {
      title: "Event emitted by schedule",
      subtitle: event.id!,
    };
  },

  getRootEventValues: (): Record<string, string> => {
    return {};
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent?: WorkflowsWorkflowEvent) => {

    const props: TriggerProps = {
      title: node.name!,
      iconSlug: trigger.icon,
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: [
        {
          icon: "calendar-cog",
          label: formatScheduleDescription(node.configuration as unknown as ScheduleConfiguration),
        },
        {
          icon: "arrow-big-right",
          label: formatNextTrigger(node.configuration as unknown as ScheduleConfiguration, node.metadata),
        }
      ],
      zeroStateText: "This schedule has not been triggered yet.",
    }

    if (lastEvent) {
      props.lastEventData = {
        title: "Event emitted by schedule",
        subtitle: lastEvent.id,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "processed",
      };
    }

    return props;
  },
};
