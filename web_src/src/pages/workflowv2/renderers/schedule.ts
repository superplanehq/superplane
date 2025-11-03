import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { convertUTCToLocalTime, formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";

type ScheduleConfigurationType = "hourly" | "daily" | "weekly";

interface ScheduleConfiguration {
  type: ScheduleConfigurationType
  minute?: number
  time?: string
  weekDay?: string
}

function formatScheduleDescription(configuration: ScheduleConfiguration): string {
  if (!configuration.type) {
    return ''
  }

  switch (configuration.type) {
    case 'hourly':
      return configuration.minute !== undefined ? `Hourly at :${configuration.minute.toString().padStart(2, '0')}` : 'Hourly'
    case 'daily':
      return configuration.time ? `Daily at ${convertUTCToLocalTime(configuration.time)}` : 'Daily'
    case 'weekly':
      const dayLabel = configuration.weekDay ? configuration.weekDay.charAt(0).toUpperCase() + configuration.weekDay.slice(1).toLowerCase() : ''
      return configuration.time && configuration.weekDay ? `${dayLabel}s at ${convertUTCToLocalTime(configuration.time)}` : 'Weekly'
    default:
      return 'Scheduled trigger'
  }
}

function calculateNextTrigger(configuration: ScheduleConfiguration): Date | null {
  const now = new Date()

  switch (configuration.type) {
    case 'hourly':
      const nextHour = new Date(now)
      nextHour.setMinutes(configuration.minute ?? 0)
      nextHour.setSeconds(0)
      nextHour.setMilliseconds(0)

      if (nextHour <= now) {
        nextHour.setHours(nextHour.getHours() + 1)
      }
      return nextHour

    case 'daily':
      if (!configuration.time) return null

      const [hours, minutes] = configuration.time.split(':').map(Number)
      const nextDay = new Date(now)
      nextDay.setUTCHours(hours)
      nextDay.setUTCMinutes(minutes)
      nextDay.setSeconds(0)
      nextDay.setMilliseconds(0)

      if (nextDay <= now) {
        nextDay.setDate(nextDay.getDate() + 1)
      }
      return nextDay

    case 'weekly':
      if (!configuration.time || !configuration.weekDay) return null

      const [weekHours, weekMinutes] = configuration.time.split(':').map(Number)
      const dayNames = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday']
      const targetDayIndex = dayNames.indexOf(configuration.weekDay.toLowerCase())

      if (targetDayIndex === -1) return null

      const nextWeek = new Date(now)
      const currentDayIndex = nextWeek.getDay()
      let daysUntilTarget = targetDayIndex - currentDayIndex

      if (daysUntilTarget < 0 || (daysUntilTarget === 0 && now.getUTCHours() * 60 + now.getUTCMinutes() >= weekHours * 60 + weekMinutes)) {
        daysUntilTarget += 7
      }

      nextWeek.setDate(nextWeek.getDate() + daysUntilTarget)
      nextWeek.setUTCHours(weekHours)
      nextWeek.setUTCMinutes(weekMinutes)
      nextWeek.setSeconds(0)
      nextWeek.setMilliseconds(0)

      return nextWeek

    default:
      return null
  }
}

function formatNextTrigger(configuration: ScheduleConfiguration): string {
  const nextTrigger = calculateNextTrigger(configuration)

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
  } catch (e) {
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

  getRootEventValues: (_: WorkflowsWorkflowEvent): Record<string, string> => {
    return {};
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: any) => {

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
          label: formatNextTrigger(node.configuration as unknown as ScheduleConfiguration),
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
