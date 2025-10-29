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

function formatNextTrigger(timestamp: string | undefined): string {
  if (!timestamp) {
    return "-"
  }

  try {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = date.getTime() - now.getTime()
    const diffMins = Math.floor(diffMs / 60000)

    if (diffMins < 0) {
      return 'Triggering soon...'
    }

    if (diffMins < 60) {
      return `Next: in ${diffMins}m`
    }

    if (diffMins < 1440) {
      return `Next: in ${Math.floor(diffMins / 60)}h`
    }

    return formatTimestampInUserTimezone(timestamp)
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
          label: formatNextTrigger(node.metadata?.nextTrigger as string),
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
