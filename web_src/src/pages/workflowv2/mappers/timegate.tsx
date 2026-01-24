import React from "react";
import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, StateFunction } from "./types";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { calcRelativeTimeFromDiff } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";

export const timeGateMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "timegate";

    return {
      iconSlug: "clock",
      iconColor: getColorClass("black"),
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0]
        ? getTimeGateEventSections(nodes, lastExecutions[0], nodeQueueItems, componentName)
        : undefined,
      includeEmptyState: !lastExecutions[0],
      specs: getTimeGateSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): React.ReactNode {
    const subtitle = getTimeGateEventSubtitle(execution, "timegate");
    return subtitle || "";
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = execution.metadata as
      | { nextValidTime?: string; pushedThroughBy?: { name?: string; email?: string }; pushedThroughAt?: string }
      | undefined;
    const state = getState("timeGate")(execution);

    if (state === "pushed through" && execution.updatedAt) {
      details["Pushed Through At"] = new Date(execution.updatedAt!).toLocaleString();
    }

    if (state === "opened" && execution.updatedAt) {
      details["Opened At"] = new Date(execution.updatedAt!).toLocaleString();
    }

    if (state === "cancelled" && execution.updatedAt) {
      details["Cancelled At"] = new Date(execution.updatedAt!).toLocaleString();
    }

    if (metadata?.nextValidTime) {
      details["Next Valid Time"] = new Date(metadata.nextValidTime).toLocaleString();
    }

    if (metadata?.pushedThroughBy) {
      details["Pushed Through By"] = metadata.pushedThroughBy.name || metadata.pushedThroughBy.email || "";
    }

    return details;
  },
};

export const TIME_GATE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  opened: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  "pushed through": {
    icon: "arrow-right",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-amber-100 dark:bg-amber-900/50",
    badgeColor: "bg-amber-500",
  },
};

export const timeGateStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "waiting";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    if (isTimeGatePushedThrough(execution)) {
      return "pushed through";
    }

    return "opened";
  }

  return "failed";
};

export const TIME_GATE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TIME_GATE_STATE_MAP,
  getState: timeGateStateFunction,
};

function getTimezoneDisplay(timezoneOffset: string): string {
  if (!timezoneOffset) {
    return "Not configured";
  }

  if (timezoneOffset === "current") {
    return "Current";
  }

  const offset = parseFloat(timezoneOffset);

  if (isNaN(offset)) {
    return "Invalid timezone";
  }

  if (offset === 0) return "GMT+0 (UTC)";
  if (offset > 0) return `GMT+${offset}`;

  return `GMT${offset}`;
}

const daysOfWeekOrder = { monday: 1, tuesday: 2, wednesday: 3, thursday: 4, friday: 5, saturday: 6, sunday: 7 };
const daysOfWeekLabels = {
  monday: "Mon",
  tuesday: "Tue",
  wednesday: "Wed",
  thursday: "Thu",
  friday: "Fri",
  saturday: "Sat",
  sunday: "Sun",
};
const monthLabels = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

function getTimeGateSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const configuration = node.configuration as Record<string, unknown>;
  const days = (configuration?.days as string[]) || [];
  const excludeDates = (configuration?.excludeDates as string[]) || [];
  const timeRange = (configuration?.timeRange as string) || "00:00-23:59";
  const timeRangeLabel = formatTimeRangeLabel(timeRange);
  const timezoneLabel = getTimezoneDisplay((configuration?.timezone as string) || "0");

  const formattedExcludeDates = excludeDates
    .map((date) => formatDayInYearLabel(date))
    .filter((date): date is string => Boolean(date));
  const excludeLabel = formattedExcludeDates.length > 0 ? formattedExcludeDates.join(", ") : "None";

  return [
    {
      title: "rule",
      tooltipTitle: "Time gate rules",
      iconSlug: "calendar",
      values: [
        {
          badges: [
            { label: "Allow:", bgColor: "bg-green-100", textColor: "text-green-800" },
            {
              label: days.length > 0 ? formatDaysLabel(days) : "Not configured",
              bgColor: "bg-gray-100",
              textColor: "text-gray-700",
            },
            { label: timeRangeLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" },
          ],
        },
        ...(excludeDates.length > 0
          ? [
              {
                badges: [
                  { label: "Exclude:", bgColor: "bg-red-100", textColor: "text-red-800" },
                  { label: excludeLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" },
                  { label: timeRangeLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" },
                ],
              },
            ]
          : []),
        {
          badges: [
            { label: "Timezone:", bgColor: "bg-gray-100", textColor: "text-gray-700" },
            { label: timezoneLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" },
          ],
        },
      ],
    },
  ];
}

function getTimeGateEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
  componentName: string,
): EventSection[] {
  const executionState = getState(componentName)(execution);
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const subtitle = getTimeGateEventSubtitle(execution, componentName);

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventState: executionState,
    eventId: execution.rootEvent!.id!,
    eventSubtitle: subtitle,
  };

  return [eventSection];
}

function getTimeGateEventSubtitle(
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): React.ReactNode | undefined {
  const executionState = getState(componentName)(execution);
  const timeAgo = execution.updatedAt
    ? formatTimeAgo(new Date(execution.updatedAt))
    : execution.createdAt
      ? formatTimeAgo(new Date(execution.createdAt))
      : "";

  if (executionState === "waiting") {
    const executionMetadata = execution.metadata as { nextValidTime?: string };
    if (executionMetadata?.nextValidTime) {
      return <TimeGateCountdown nextValidTime={executionMetadata.nextValidTime} timeAgo={timeAgo} />;
    }
  }

  return timeAgo || undefined;
}

const TimeGateCountdown: React.FC<{ nextValidTime: string; timeAgo?: string }> = ({ nextValidTime, timeAgo }) => {
  const nextRunTime = React.useMemo(() => new Date(nextValidTime), [nextValidTime]);
  const [timeLeft, setTimeLeft] = React.useState<number>(() => nextRunTime.getTime() - Date.now());

  React.useEffect(() => {
    if (Number.isNaN(nextRunTime.getTime())) {
      return;
    }

    const update = () => {
      setTimeLeft(nextRunTime.getTime() - Date.now());
    };

    update();
    const interval = setInterval(update, 1000);
    return () => clearInterval(interval);
  }, [nextRunTime]);

  if (Number.isNaN(nextRunTime.getTime())) {
    return <span>{timeAgo || ""}</span>;
  }

  const timeLeftText = timeLeft > 0 ? calcRelativeTimeFromDiff(timeLeft) : "Ready to run";
  return (
    <span>
      Runs in {timeLeftText}
      {timeAgo ? ` · ${timeAgo}` : ""}
    </span>
  );
};

function isTimeGatePushedThrough(execution: WorkflowsWorkflowNodeExecution): boolean {
  if (!execution.updatedAt) {
    return false;
  }

  const metadata = execution.metadata as { nextValidTime?: string } | undefined;
  if (!metadata?.nextValidTime) {
    return false;
  }

  const nextValidTime = new Date(metadata.nextValidTime);
  const finishedAt = new Date(execution.updatedAt);
  if (Number.isNaN(nextValidTime.getTime()) || Number.isNaN(finishedAt.getTime())) {
    return false;
  }

  return nextValidTime.getTime() - finishedAt.getTime() > 1000;
}

function formatDaysLabel(days: string[]): string {
  return days
    .slice()
    .sort(
      (a, b) =>
        daysOfWeekOrder[a.trim() as keyof typeof daysOfWeekOrder] -
        daysOfWeekOrder[b.trim() as keyof typeof daysOfWeekOrder],
    )
    .map((day) => daysOfWeekLabels[day.trim() as keyof typeof daysOfWeekLabels] || day.trim())
    .join(", ");
}

function formatTimeRangeLabel(timeRange: string): string {
  const { start, end } = parseTimeRange(timeRange);
  if (start === "00:00" && end === "23:59") {
    return "all day";
  }

  return `${start} - ${end}`;
}

function parseTimeRange(timeRange: string): { start: string; end: string } {
  if (!timeRange) {
    return { start: "00:00", end: "23:59" };
  }

  const parts = timeRange.split("-");
  if (parts.length !== 2) {
    return { start: "00:00", end: "23:59" };
  }

  const start = parts[0].trim() || "00:00";
  const end = parts[1].trim() || "23:59";
  return { start, end };
}

function formatDayInYearLabel(dayInYear: string): string | null {
  const match = dayInYear.match(/^(\d{1,2})\/(\d{1,2})$/);
  if (!match) return null;

  const month = Number.parseInt(match[1], 10);
  const day = Number.parseInt(match[2], 10);
  if (Number.isNaN(month) || Number.isNaN(day) || month < 1 || month > 12 || day < 1 || day > 31) {
    return null;
  }

  return `${monthLabels[month - 1]} ${day}`;
}
