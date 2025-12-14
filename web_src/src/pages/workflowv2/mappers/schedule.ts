import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";

type ScheduleConfigurationType = "minutes" | "hours" | "days" | "weeks" | "months" | "cron";

interface ScheduleConfiguration {
  type: ScheduleConfigurationType;
  minutesInterval?: number;
  hoursInterval?: number;
  daysInterval?: number;
  weeksInterval?: number;
  monthsInterval?: number;
  minute?: number;
  hour?: number;
  weekDays?: string[];
  dayOfMonth?: number;
  cronExpression?: string;
  timezone?: string;
}

function formatScheduleDescription(configuration: ScheduleConfiguration): string {
  if (!configuration.type) {
    return "";
  }

  switch (configuration.type) {
    case "minutes": {
      return configuration.minutesInterval !== undefined
        ? `Every ${configuration.minutesInterval} minute${configuration.minutesInterval === 1 ? "" : "s"}`
        : "Every X minutes";
    }
    case "hours": {
      const interval = configuration.hoursInterval || 1;
      const minute = configuration.minute || 0;
      return `Every ${interval} hour${interval === 1 ? "" : "s"} at :${minute.toString().padStart(2, "0")}`;
    }
    case "days": {
      const interval = configuration.daysInterval || 1;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;
      const time = `${hour.toString().padStart(2, "0")}:${minute.toString().padStart(2, "0")}`;
      return `Every ${interval} day${interval === 1 ? "" : "s"} at ${time}`;
    }
    case "weeks": {
      const interval = configuration.weeksInterval || 1;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;
      const time = `${hour.toString().padStart(2, "0")}:${minute.toString().padStart(2, "0")}`;
      const weekDays = configuration.weekDays || ["monday"];
      const dayLabels = weekDays.map(day =>
        day.charAt(0).toUpperCase() + day.slice(1).toLowerCase()
      ).join(", ");
      return `Every ${interval} week${interval === 1 ? "" : "s"} on ${dayLabels} at ${time}`;
    }
    case "months": {
      const interval = configuration.monthsInterval || 1;
      const dayOfMonth = configuration.dayOfMonth || 1;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;
      const time = `${hour.toString().padStart(2, "0")}:${minute.toString().padStart(2, "0")}`;
      return `Every ${interval} month${interval === 1 ? "" : "s"} on day ${dayOfMonth} at ${time}`;
    }
    case "cron": {
      return configuration.cronExpression
        ? `Cron: ${configuration.cronExpression}`
        : "Custom cron schedule";
    }
    default:
      return "Scheduled trigger";
  }
}

function calculateNextTrigger(configuration: ScheduleConfiguration, referenceNextTrigger?: string): Date | null {
  // Always use backend-calculated nextTrigger first if available
  if (referenceNextTrigger) {
    try {
      console.log(referenceNextTrigger)
      return new Date(referenceNextTrigger);
    } catch {
      // Fall through to frontend calculation if parsing fails
    }
  }

  const now = new Date();

  // Apply timezone offset if specified
  const timezoneOffset = configuration.timezone ? parseFloat(configuration.timezone) : 0;
  const nowInTZ = new Date(now.getTime() + timezoneOffset * 60 * 60 * 1000);

  switch (configuration.type) {
    case "minutes": {
      if (configuration.minutesInterval === undefined) return null;

      const interval = configuration.minutesInterval;

      // Fallback calculation when no backend reference is available
      const nextTrigger = new Date(nowInTZ);
      nextTrigger.setSeconds(0);
      nextTrigger.setMilliseconds(0);
      nextTrigger.setMinutes(nextTrigger.getMinutes() + interval);
      return nextTrigger;
    }

    case "hours": {
      const interval = configuration.hoursInterval || 1;
      const minute = configuration.minute || 0;

      const nextTrigger = new Date(nowInTZ);
      nextTrigger.setMinutes(minute);
      nextTrigger.setSeconds(0);
      nextTrigger.setMilliseconds(0);

      // Add the interval
      nextTrigger.setHours(nextTrigger.getHours() + interval);

      return new Date(nextTrigger.getTime() - timezoneOffset * 60 * 60 * 1000);
    }

    case "days": {
      const interval = configuration.daysInterval || 1;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      const nextTrigger = new Date(nowInTZ);
      nextTrigger.setHours(hour);
      nextTrigger.setMinutes(minute);
      nextTrigger.setSeconds(0);
      nextTrigger.setMilliseconds(0);

      // Add the interval
      nextTrigger.setDate(nextTrigger.getDate() + interval);

      return new Date(nextTrigger.getTime() - timezoneOffset * 60 * 60 * 1000);
    }

    case "weeks": {
      const interval = configuration.weeksInterval || 1;
      const weekDays = configuration.weekDays || ["monday"];
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      const dayNames = ["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"];
      const validDayIndices = weekDays
        .map(day => dayNames.indexOf(day.toLowerCase()))
        .filter(index => index !== -1);

      if (validDayIndices.length === 0) return null;

      const nextTrigger = new Date(nowInTZ);
      nextTrigger.setHours(hour);
      nextTrigger.setMinutes(minute);
      nextTrigger.setSeconds(0);
      nextTrigger.setMilliseconds(0);

      // Add the interval in weeks and find next valid day
      nextTrigger.setDate(nextTrigger.getDate() + interval * 7);

      // Find the closest valid weekday
      for (let i = 0; i < 7; i++) {
        const testDate = new Date(nextTrigger);
        testDate.setDate(testDate.getDate() + i);
        if (validDayIndices.includes(testDate.getDay())) {
          testDate.setHours(hour);
          testDate.setMinutes(minute);
          return new Date(testDate.getTime() - timezoneOffset * 60 * 60 * 1000);
        }
      }

      return null;
    }

    case "months": {
      const interval = configuration.monthsInterval || 1;
      const dayOfMonth = configuration.dayOfMonth || 1;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      const nextTrigger = new Date(nowInTZ);
      nextTrigger.setMonth(nextTrigger.getMonth() + interval);
      nextTrigger.setDate(dayOfMonth);
      nextTrigger.setHours(hour);
      nextTrigger.setMinutes(minute);
      nextTrigger.setSeconds(0);
      nextTrigger.setMilliseconds(0);

      return new Date(nextTrigger.getTime() - timezoneOffset * 60 * 60 * 1000);
    }

    case "cron": {
      // For cron expressions, we can't easily calculate in frontend
      // The backend handles the actual cron scheduling
      return null;
    }

    default:
      return null;
  }
}

function formatNextTrigger(configuration: ScheduleConfiguration, metadata?: { nextTrigger?: string }): string {
  const nextTrigger = calculateNextTrigger(
    configuration,
    metadata?.nextTrigger,
  );

  if (!nextTrigger) {
    return "-";
  }

  try {
    const now = new Date();
    const diffMs = nextTrigger.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins <= 0) {
      return "Triggering soon...";
    }

    if (diffMins < 60) {
      return `Next: in ${diffMins}m`;
    }

    if (diffMins < 1440) {
      return `Next: in ${Math.floor(diffMins / 60)}h`;
    }

    return formatTimestampInUserTimezone(nextTrigger.toISOString());
  } catch {
    return "";
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
        },
      ],
      zeroStateText: "This schedule has not been triggered yet.",
    };

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
