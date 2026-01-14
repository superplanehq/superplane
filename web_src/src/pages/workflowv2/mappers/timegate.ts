import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { calcRelativeTimeFromDiff } from "@/lib/utils";

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
      eventStateMap: getStateMap(componentName),
    };
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
  componentName: string,
): EventSection[] {
  const executionState = getState(componentName)(execution);
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  let subtitle: string | undefined;

  // If running, show next run time in the subtitle
  if (executionState === "running") {
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
