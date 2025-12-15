import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { getNextCronExecution } from "@/utils/cron";
import { TriggerRenderer, CustomFieldRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";
import React from "react";

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
      const dayLabels = weekDays.map((day) => day.charAt(0).toUpperCase() + day.slice(1).toLowerCase()).join(", ");
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
      return configuration.cronExpression ? `Cron: ${configuration.cronExpression}` : "Custom cron schedule";
    }
    default:
      return "Scheduled trigger";
  }
}

function calculateNextTrigger(configuration: ScheduleConfiguration, referenceNextTrigger?: string): Date | null {
  // Always use backend-calculated nextTrigger first if available
  if (referenceNextTrigger) {
    try {
      return new Date(referenceNextTrigger);
    } catch {
      // Fall through to frontend calculation if parsing fails
    }
  }

  if (!configuration.type) return null;

  const now = new Date();

  // Parse timezone like the Go backend - create location from offset
  const timezoneOffset = configuration.timezone ? parseFloat(configuration.timezone) : 0;
  const timezoneOffsetMs = timezoneOffset * 60 * 60 * 1000;

  // Convert current time to the target timezone
  const nowInTZ = new Date(now.getTime() + timezoneOffsetMs);

  switch (configuration.type) {
    case "minutes": {
      if (
        configuration.minutesInterval === undefined ||
        configuration.minutesInterval < 1 ||
        configuration.minutesInterval > 59
      )
        return null;

      const interval = configuration.minutesInterval;

      // Simulate reference time logic from Go - for minutes we need a reference point
      // Since we don't have referenceTime in frontend, use current time as reference
      const reference = nowInTZ;
      const minutesElapsed = Math.floor((nowInTZ.getTime() - reference.getTime()) / (60 * 1000));

      const completedIntervals = Math.floor(Math.max(0, minutesElapsed) / interval);
      const nextTriggerMinutes = (completedIntervals + 1) * interval;

      const nextTriggerInTZ = new Date(reference.getTime() + nextTriggerMinutes * 60 * 1000);

      // If nextTrigger is in the past or now, add another interval
      if (nextTriggerInTZ <= nowInTZ) {
        nextTriggerInTZ.setTime(nextTriggerInTZ.getTime() + interval * 60 * 1000);
      }

      return new Date(nextTriggerInTZ.getTime());
    }

    case "hours": {
      if (
        configuration.hoursInterval === undefined ||
        configuration.hoursInterval < 1 ||
        configuration.hoursInterval > 23
      )
        return null;

      const interval = configuration.hoursInterval;
      const minute = configuration.minute || 0;

      if (minute < 0 || minute > 59) return null;

      // Match Go backend: start with current time in timezone + interval, set minute
      const nextTriggerInTZ = new Date(nowInTZ);
      nextTriggerInTZ.setHours(nextTriggerInTZ.getHours() + interval);
      nextTriggerInTZ.setMinutes(minute);
      nextTriggerInTZ.setSeconds(0);
      nextTriggerInTZ.setMilliseconds(0);

      return new Date(nextTriggerInTZ.getTime());
    }

    case "days": {
      if (configuration.daysInterval === undefined || configuration.daysInterval < 1 || configuration.daysInterval > 31)
        return null;

      const interval = configuration.daysInterval;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;

      // Match Go backend: add interval days in timezone, set time
      const nextTriggerInTZ = new Date(nowInTZ);
      nextTriggerInTZ.setDate(nextTriggerInTZ.getDate() + interval);
      nextTriggerInTZ.setHours(hour);
      nextTriggerInTZ.setMinutes(minute);
      nextTriggerInTZ.setSeconds(0);
      nextTriggerInTZ.setMilliseconds(0);

      return new Date(nextTriggerInTZ.getTime());
    }

    case "weeks": {
      if (
        configuration.weeksInterval === undefined ||
        configuration.weeksInterval < 1 ||
        configuration.weeksInterval > 52
      )
        return null;
      if (!configuration.weekDays || configuration.weekDays.length === 0) return null;

      const interval = configuration.weeksInterval;
      const weekDays = configuration.weekDays;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;

      const dayNames = ["sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"];
      const validDayIndices = new Set();

      for (const dayStr of weekDays) {
        const dayIndex = dayNames.indexOf(dayStr.toLowerCase());
        if (dayIndex !== -1) {
          validDayIndices.add(dayIndex);
        }
      }

      if (validDayIndices.size === 0) return null;

      // Match Go backend: add interval weeks in timezone, then find first valid weekday
      const nextIntervalStart = new Date(nowInTZ);
      nextIntervalStart.setDate(nextIntervalStart.getDate() + interval * 7);

      // Go to start of week (Sunday = 0)
      const daysToSubtract = nextIntervalStart.getDay();
      nextIntervalStart.setDate(nextIntervalStart.getDate() - daysToSubtract);

      // Find first valid weekday in that week
      for (let i = 0; i < 7; i++) {
        const checkDate = new Date(nextIntervalStart);
        checkDate.setDate(checkDate.getDate() + i);
        if (validDayIndices.has(checkDate.getDay())) {
          checkDate.setHours(hour);
          checkDate.setMinutes(minute);
          checkDate.setSeconds(0);
          checkDate.setMilliseconds(0);
          return new Date(checkDate.getTime());
        }
      }

      return null;
    }

    case "months": {
      if (
        configuration.monthsInterval === undefined ||
        configuration.monthsInterval < 1 ||
        configuration.monthsInterval > 24
      )
        return null;
      if (configuration.dayOfMonth === undefined || configuration.dayOfMonth < 1 || configuration.dayOfMonth > 31)
        return null;

      const interval = configuration.monthsInterval;
      const dayOfMonth = configuration.dayOfMonth;
      const hour = configuration.hour || 0;
      const minute = configuration.minute || 0;

      if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;

      // Match Go backend: add interval months in timezone, set day/hour/minute
      const nextTriggerInTZ = new Date(nowInTZ);
      nextTriggerInTZ.setMonth(nextTriggerInTZ.getMonth() + interval);
      nextTriggerInTZ.setDate(dayOfMonth);
      nextTriggerInTZ.setHours(hour);
      nextTriggerInTZ.setMinutes(minute);
      nextTriggerInTZ.setSeconds(0);
      nextTriggerInTZ.setMilliseconds(0);

      return new Date(nextTriggerInTZ.getTime());
    }

    case "cron": {
      if (!configuration.cronExpression) return null;

      try {
        const nextTime = getNextCronExecution(configuration.cronExpression, nowInTZ);

        if (!nextTime) return null;

        return new Date(nextTime.getTime());
      } catch {
        return null;
      }
    }

    default:
      return null;
  }
}

function formatNextTrigger(configuration: ScheduleConfiguration, metadata?: { nextTrigger?: string }): string {
  const nextTrigger = calculateNextTrigger(configuration, metadata?.nextTrigger);

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

/**
 * Custom field renderer for schedule trigger configuration
 */
export const scheduleCustomFieldRenderer: CustomFieldRenderer = {
  render: (_node: ComponentsNode, configuration: Record<string, unknown>) => {
    const scheduleConfig = configuration as unknown as ScheduleConfiguration;
    const scheduleDescription = formatScheduleDescription(scheduleConfig);
    const nextTrigger = formatNextTrigger(scheduleConfig, undefined);

    return React.createElement(
      "div",
      { className: "border-t-1 border-gray-200" },
      React.createElement(
        "div",
        { className: "space-y-3" },
        React.createElement(
          "div",
          null,
          React.createElement(
            "span",
            { className: "text-sm font-medium text-gray-700 dark:text-gray-300" },
            "Runs on:",
          ),
          React.createElement(
            "div",
            { className: "text-sm text-gray-900 dark:text-gray-100 mt-1 border-1 p-2 bg-zinc-100" },
            scheduleDescription || "Schedule not configured",
          ),
        ),
        React.createElement(
          "div",
          null,
          React.createElement(
            "span",
            { className: "text-sm font-medium text-gray-700 dark:text-gray-300" },
            "Next run:",
          ),
          React.createElement(
            "div",
            { className: "text-sm text-gray-900 dark:text-gray-100 mt-1 border-1 p-2 bg-zinc-100" },
            nextTrigger,
          ),
        ),
      ),
    );
  },
};
