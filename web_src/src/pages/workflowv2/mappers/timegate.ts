import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { MetadataItem } from "@/ui/metadataList";
import { getTriggerRenderer } from ".";
import { calcRelativeTimeFromDiff } from "@/lib/utils";

export const timeGateMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecution: WorkflowsWorkflowNodeExecution | null,
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    return {
      iconSlug: "clock",
      headerColor: getBackgroundColorClass(componentDefinition?.color || "blue"),
      iconColor: getColorClass(componentDefinition?.color || "blue"),
      iconBackground: getBackgroundColorClass(componentDefinition?.color || "blue"),
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getTimeGateEventSections(nodes, lastExecution, nodeQueueItems),
      metadata: getTimeGateMetadataList(node),
      specs: getTimeGateSpecs(node),
    };
  },
};

function getTimeGateMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  // Get mode
  const mode = configuration?.mode || "include_range";
  const getModeLabel = (mode: string) => {
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
  };

  metadata.push({
    icon: "settings",
    label: getModeLabel(mode),
  });

  // Get time window
  let startTime = "00:00";
  let endTime = "23:59";

  if (mode === "include_specific" || mode === "exclude_specific") {
    startTime = `${configuration.startDayInYear} ${configuration.startTime}`;
    endTime = `${configuration.endDayInYear} ${configuration.endTime}`;
  } else {
    startTime = `${configuration.startTime}`;
    endTime = `${configuration.endTime}`;
  }

  const timeWindow = `${startTime} - ${endTime}`;
  metadata.push({
    icon: "clock",
    label: timeWindow,
  });

  // Get timezone
  const timezone = configuration?.timezone || "0";
  const getTimezoneDisplay = (timezoneOffset: string) => {
    const offset = parseFloat(timezoneOffset);
    if (offset === 0) return "GMT+0 (UTC)";
    if (offset > 0) return `GMT+${offset}`;
    return `GMT${offset}`; // Already has the minus sign
  };
  const timezoneDisplay = getTimezoneDisplay(timezone);

  metadata.push({
    icon: "globe",
    label: `Timezone: ${timezoneDisplay}`,
  });

  return metadata;
}

const daysOfWeekOrder = { monday: 1, tuesday: 2, wednesday: 3, thursday: 4, friday: 5, saturday: 6, sunday: 7 };

function getTimeGateSpecs(node: ComponentsNode): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as any;
  const days = configuration?.days || [];

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

function getRunItemState(execution: WorkflowsWorkflowNodeExecution): "success" | "failed" | "running" {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

function getTimeGateEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution | null,
  nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
): EventSection[] {
  const sections: EventSection[] = [];

  // Add Last Event section
  if (!execution) {
    sections.push({
      title: "LAST EVENT",
      eventTitle: "No events received yet",
      eventState: "neutral" as const,
    });
  } else {
    const executionState = getRunItemState(execution);
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

    sections.push({
      title: "LAST EVENT",
      subtitle: subtitle,
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: executionState,
    });
  }

  // Add Next in Queue section if there are queued items
  if (nodeQueueItems && nodeQueueItems.length > 0) {
    const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
    const rootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    if (queueItem.rootEvent) {
      const { title } = rootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
      sections.push({
        title: "NEXT IN QUEUE",
        receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
        eventTitle: title,
        eventState: "next-in-queue" as const,
      });
    }
  }

  return sections;
}
