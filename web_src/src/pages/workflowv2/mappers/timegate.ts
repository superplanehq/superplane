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

function getTimeGateMetadataList(node: ComponentsNode): MetadataItem[] {
  const configuration = node.configuration as Record<string, unknown>;
  const mode = configuration?.mode;

  return [
    {
      icon: "settings",
      label: getTimeGateModeLabel(mode as string),
    },
    {
      icon: "clock",
      label: getTimeWindow(mode as string, configuration),
    },
    {
      icon: "globe",
      label: `Timezone: ${getTimezoneDisplay((configuration?.timezone as string) || "0")}`,
    },
  ];
}

function getTimeGateModeLabel(mode: string): string {
  if (!mode) {
    return "Not configured";
  }

  switch (mode) {
    case "include_range":
      return "Include Range";
    case "exclude_range":
      return "Exclude Range";
    case "include_specific":
      return "Include Specific";
    case "exclude_specific":
      return "Exclude Specific";
    default:
      return mode.charAt(0).toUpperCase() + mode.slice(1).replace(/_/g, " ");
  }
}

function getTimeWindow(mode: string, configuration: Record<string, unknown>): string {
  let startTime = "00:00";
  let endTime = "23:59";

  // Handle undefined mode or configuration values
  if (!mode) {
    return "Not configured";
  }

  if (mode === "include_specific" || mode === "exclude_specific") {
    const startDay = configuration.startDayInYear || "Day 1";
    const startTimeVal = configuration.startTime || "00:00";
    const endDay = configuration.endDayInYear || "Day 365";
    const endTimeVal = configuration.endTime || "23:59";
    startTime = `${startDay} ${startTimeVal}`;
    endTime = `${endDay} ${endTimeVal}`;
  } else {
    startTime = `${configuration.startTime || "00:00"}`;
    endTime = `${configuration.endTime || "23:59"}`;
  }

  return `${startTime} - ${endTime}`;
}

function getTimezoneDisplay(timezoneOffset: string): string {
  // Handle undefined or invalid timezone
  if (!timezoneOffset) {
    return "Not configured";
  }

  const offset = parseFloat(timezoneOffset);

  // Handle invalid number
  if (isNaN(offset)) {
    return "Invalid timezone";
  }

  if (offset === 0) return "GMT+0 (UTC)";
  if (offset > 0) return `GMT+${offset}`;

  //
  // Already has the minus sign
  //
  return `GMT${offset}`;
}

const daysOfWeekOrder = { monday: 1, tuesday: 2, wednesday: 3, thursday: 4, friday: 5, saturday: 6, sunday: 7 };

function getTimeGateSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as Record<string, unknown>;
  const days = (configuration?.days as string[]) || [];

  if (days && days.length > 0) {
    specs.push({
      title: "day",
      tooltipTitle: "Days of the week",
      iconSlug: "calendar",
      values: [
        ...days
          .sort(
            (a: string, b: string) =>
              daysOfWeekOrder[a.trim() as keyof typeof daysOfWeekOrder] -
              daysOfWeekOrder[b.trim() as keyof typeof daysOfWeekOrder],
          )
          .map((day: string) => ({
            badges: [
              {
                label: day.trim(),
                bgColor: "bg-gray-100",
                textColor: "text-gray-700",
              },
            ],
          })),
      ],
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
