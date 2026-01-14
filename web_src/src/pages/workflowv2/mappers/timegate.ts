import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, StateFunction } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState, EventStateMap, DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer, getState } from ".";
import { calcRelativeTimeFromDiff } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";

export const TIMEGATE_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
};

export const timeGateStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
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
    return "success";
  }

  return "failed";
};

export const TIMEGATE_STATE_REGISTRY: EventStateRegistry = {
  stateMap: TIMEGATE_STATE_MAP,
  getState: timeGateStateFunction,
};

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
      headerColor: "bg-white",
      iconColor: getColorClass("black"),
      iconBackground: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecutions[0]
        ? getTimeGateEventSections(nodes, lastExecutions[0], nodeQueueItems, componentName)
        : undefined,
      includeEmptyState: !lastExecutions[0],
      metadata: getTimeGateMetadataList(node),
      specs: getTimeGateSpecs(node),
      eventStateMap: TIMEGATE_STATE_MAP,
    };
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};
    const metadata = execution.metadata as Record<string, unknown> | undefined;

    if (execution.createdAt) {
      details["Started at"] = new Date(execution.createdAt).toLocaleString();
    }

    if (execution.state === "STATE_FINISHED" && execution.updatedAt) {
      details["Finished at"] = new Date(execution.updatedAt).toLocaleString();
    }

    details["Timeline"] = buildTimeGateTimeline(execution, metadata);

    return details;
  },
};

function getTimeGateMetadataList(_node: ComponentsNode): MetadataItem[] {
  // Metadata is now shown via specs tooltip
  return [];
}

const daysOfWeekOrder = { monday: 1, tuesday: 2, wednesday: 3, thursday: 4, friday: 5, saturday: 6, sunday: 7 };
const dayAbbreviations: Record<string, string> = {
  monday: "Mon",
  tuesday: "Tue",
  wednesday: "Wed",
  thursday: "Thu",
  friday: "Fri",
  saturday: "Sat",
  sunday: "Sun",
};

const monthNames: Record<number, string> = {
  1: "Jan", 2: "Feb", 3: "Mar", 4: "Apr", 5: "May", 6: "Jun",
  7: "Jul", 8: "Aug", 9: "Sep", 10: "Oct", 11: "Nov", 12: "Dec",
};

function formatDays(days: string[]): string {
  if (!days || days.length === 0) return "";
  
  const sortedDays = [...days].sort(
    (a, b) => daysOfWeekOrder[a as keyof typeof daysOfWeekOrder] - daysOfWeekOrder[b as keyof typeof daysOfWeekOrder]
  );
  
  return sortedDays.map(d => dayAbbreviations[d] || d).join(", ");
}

function formatTime(startTime: string, endTime: string): string {
  if (startTime === "00:00" && endTime === "23:59") {
    return "all day";
  }
  return `${startTime}-${endTime}`;
}

function formatDate(dateStr: string): string {
  // Parse MM-DD format
  const match = dateStr.match(/^(\d{2})-(\d{2})$/);
  if (match) {
    const month = parseInt(match[1], 10);
    const day = parseInt(match[2], 10);
    return `${monthNames[month] || match[1]} ${day}`;
  }
  return dateStr;
}

function getTimeGateSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as Record<string, unknown>;
  const items = (configuration?.items as Array<Record<string, unknown>>) || [];
  const excludeDates = (configuration?.exclude_dates as Array<Record<string, unknown>>) || [];

  const ruleValues: Array<{ badges: Array<{ label: string; bgColor: string; textColor: string }> }> = [];

  // Add time window rules
  items.forEach((item) => {
    const days = (item.days as string[]) || [];
    const startTime = (item.startTime as string) || "00:00";
    const endTime = (item.endTime as string) || "23:59";
    
    const daysStr = formatDays(days);
    const timeStr = formatTime(startTime, endTime);
    
    ruleValues.push({
      badges: [
        {
          label: "Allow",
          bgColor: "bg-green-100",
          textColor: "text-green-700",
        },
        {
          label: daysStr || "No days",
          bgColor: "bg-gray-100",
          textColor: "text-gray-700",
        },
        {
          label: timeStr,
          bgColor: "bg-gray-100",
          textColor: "text-gray-700",
        },
      ],
    });
  });

  // Add exclude date rules
  excludeDates.forEach((excludeDate) => {
    const date = (excludeDate.date as string) || "";
    const startTime = (excludeDate.startTime as string) || "00:00";
    const endTime = (excludeDate.endTime as string) || "23:59";
    
    const dateStr = formatDate(date);
    const timeStr = formatTime(startTime, endTime);
    
    ruleValues.push({
      badges: [
        {
          label: "Exclude",
          bgColor: "bg-red-100",
          textColor: "text-red-700",
        },
        {
          label: dateStr,
          bgColor: "bg-gray-100",
          textColor: "text-gray-700",
        },
        {
          label: timeStr,
          bgColor: "bg-gray-100",
          textColor: "text-gray-700",
        },
      ],
    });
  });

  if (ruleValues.length > 0) {
    specs.push({
      title: "rule",
      tooltipTitle: "Time gate rules",
      iconSlug: "clock",
      values: ruleValues,
    });
  }

  return specs;
}

function getTimeGateEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
  _componentName: string,
): EventSection[] {
  const executionState = timeGateStateFunction(execution);
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  let subtitle: string | undefined;

  // If waiting, show next run time in the subtitle
  if (executionState === "waiting") {
    const executionMetadata = execution.metadata as { nextValidTime?: string };
    if (executionMetadata?.nextValidTime) {
      const nextRunTime = new Date(executionMetadata.nextValidTime);
      const now = new Date();
      const timeDiff = nextRunTime.getTime() - now.getTime();
      const timeLeftText = timeDiff > 0 ? calcRelativeTimeFromDiff(timeDiff) : "Ready to run";
      subtitle = `Runs in ${timeLeftText}`;
    }
  }

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventState: executionState,
    eventId: execution.rootEvent?.id,
    eventSubtitle: subtitle,
  };

  return [eventSection];
}

function buildTimeGateTimeline(
  execution: WorkflowsWorkflowNodeExecution,
  metadata?: Record<string, unknown>,
): Array<{ label: string; status: string; timestamp?: string; comment?: string }> {
  const timeline: Array<{ label: string; status: string; timestamp?: string; comment?: string }> = [];

  // Started event
  if (execution.createdAt) {
    timeline.push({
      label: "Execution started",
      status: "Started",
      timestamp: formatTimeAgo(new Date(execution.createdAt)),
    });
  }

  // Waiting for time window (if running and has nextValidTime)
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_PENDING") {
    const nextValidTime = metadata?.nextValidTime as string | undefined;
    if (nextValidTime) {
      try {
        const nextTime = new Date(nextValidTime);
        const now = new Date();
        const timeDiff = nextTime.getTime() - now.getTime();
        const timeLeftText = timeDiff > 0 ? calcRelativeTimeFromDiff(timeDiff) : "Ready to run";
        timeline.push({
          label: "Waiting for time window",
          status: "Waiting",
          timestamp: `runs in ${timeLeftText}`,
        });
      } catch {
        // Invalid date, skip
      }
    }
  }

  // Cancelled event
  if (execution.result === "RESULT_CANCELLED") {
    const cancelledBy = metadata?.cancelledBy as
      | { at?: string; userId?: string; email?: string; name?: string }
      | undefined;

    if (cancelledBy) {
      const userDisplayName = cancelledBy.name || cancelledBy.email || "Unknown user";
      const cancelledAt = cancelledBy.at ? formatTimeAgo(new Date(cancelledBy.at)) : (execution.updatedAt ? formatTimeAgo(new Date(execution.updatedAt)) : "");
      timeline.push({
        label: "Cancelled",
        status: "Cancelled",
        timestamp: cancelledAt ? `${cancelledAt} by ${userDisplayName}` : `by ${userDisplayName}`,
      });
    } else if (execution.updatedAt) {
      timeline.push({
        label: "Cancelled",
        status: "Cancelled",
        timestamp: formatTimeAgo(new Date(execution.updatedAt)),
      });
    }
  }
  // Finished event
  else if (execution.state === "STATE_FINISHED") {
    if (execution.updatedAt) {
      // Check if it was pushed through manually
      const pushedThrough = metadata?.pushedThrough as
        | { at?: string; userId?: string; email?: string; name?: string }
        | undefined;

      if (pushedThrough) {
        // Manually pushed through
        const userDisplayName = pushedThrough.name || pushedThrough.email || "Unknown user";
        const pushedAt = pushedThrough.at ? formatTimeAgo(new Date(pushedThrough.at)) : formatTimeAgo(new Date(execution.updatedAt));
        timeline.push({
          label: "Manually pushed through",
          status: "Passed",
          timestamp: `${pushedAt} by ${userDisplayName}`,
        });
      } else {
        // Natural time window reached
        const result = execution.result === "RESULT_PASSED" ? "Time window reached" : "Finished";
        timeline.push({
          label: result,
          status: execution.result === "RESULT_PASSED" ? "Passed" : "Finished",
          timestamp: formatTimeAgo(new Date(execution.updatedAt)),
        });
      }
    }
  }

  // Sort by actual timestamp (not formatted string) for proper ordering
  timeline.sort((a, b) => {
    let aTime: number | null = null;
    let bTime: number | null = null;

    if (execution.createdAt && a.label === "Execution started") {
      aTime = new Date(execution.createdAt).getTime();
    } else if (execution.updatedAt && (a.label === "Time window reached" || a.label === "Finished")) {
      aTime = new Date(execution.updatedAt).getTime();
    } else if (metadata?.nextValidTime && a.label === "Waiting for time window") {
      try {
        aTime = new Date(metadata.nextValidTime as string).getTime();
      } catch {
        // Invalid date
      }
    }

    if (execution.createdAt && b.label === "Execution started") {
      bTime = new Date(execution.createdAt).getTime();
    } else if (execution.updatedAt && (b.label === "Time window reached" || b.label === "Finished")) {
      bTime = new Date(execution.updatedAt).getTime();
    } else if (metadata?.nextValidTime && b.label === "Waiting for time window") {
      try {
        bTime = new Date(metadata.nextValidTime as string).getTime();
      } catch {
        // Invalid date
      }
    }

    if (aTime === null && bTime === null) return 0;
    if (aTime === null) return 1;
    if (bTime === null) return -1;
    return aTime - bTime;
  });

  return timeline;
}
