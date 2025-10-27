import { ComponentsNode, TriggersTrigger } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { convertUTCToLocalTime, formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "./types";

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

/**
 * Renderer for the "schedule" trigger type
 */
export const scheduleTriggerRenderer: TriggerRenderer = {
  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger) => ({
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
        label: formatTimestampInUserTimezone(node.metadata?.nextTrigger as string),
      }
    ],
    zeroStateText: "This schedule has not been triggered yet.",
  }),
};
