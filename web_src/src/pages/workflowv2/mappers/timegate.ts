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
  const whenToRun = configuration?.when_to_run as string;
  const mode = configuration?.mode as string;

  return [
    {
      icon: "settings",
      label: getTimeGateModeLabel(whenToRun, mode),
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

function getTimeGateModeLabel(whenToRun: string | undefined, mode: string | undefined): string {
  if (whenToRun && whenToRun !== "custom") {
    switch (whenToRun) {
      case "template_working_hours":
        return "Run during working hours";
      case "template_outside_working_hours":
        return "Run outside of working hours";
      case "template_weekends":
        return "Run on weekends";
      case "template_no_weekends":
        return "Don't run on weekends";
      default:
        return whenToRun.charAt(0).toUpperCase() + whenToRun.slice(1).replace(/_/g, " ");
    }
  }

  if (!mode) {
    return "Not configured";
  }

  switch (mode) {
    case "include":
      return "Include";
    case "exclude":
      return "Exclude";
    default:
      return mode.charAt(0).toUpperCase() + mode.slice(1).replace(/_/g, " ");
  }
}

function getTimeWindow(_mode: string, configuration: Record<string, unknown>): string {
  const items = (configuration?.items as Array<Record<string, unknown>>) || [];

  if (items.length === 0) {
    return "Not configured";
  }

  // Display summary of items
  if (items.length === 1) {
    const item = items[0];
    const itemType = item.type as string;
    const startTime = (item.startTime as string) || "00:00";
    const endTime = (item.endTime as string) || "23:59";

    if (itemType === "specific_dates") {
      const startDay = (item.startDayInYear as string) || "";
      const endDay = (item.endDayInYear as string) || "";
      return `${startDay} ${startTime} - ${endDay} ${endTime}`;
    } else {
      return `${startTime} - ${endTime}`;
    }
  }

  return `${items.length} time window${items.length > 1 ? "s" : ""}`;
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
  const items = (configuration?.items as Array<Record<string, unknown>>) || [];

  // Collect all days from weekly items
  const allDays = new Set<string>();
  items.forEach((item) => {
    if (item.type === "weekly") {
      const days = (item.days as string[]) || [];
      days.forEach((day) => allDays.add(day));
    }
  });

  if (allDays.size > 0) {
    specs.push({
      title: "day",
      tooltipTitle: "Days of the week",
      iconSlug: "calendar",
      values: [
        ...Array.from(allDays)
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

  // Add count of items
  if (items.length > 0) {
    specs.push({
      title: "items",
      tooltipTitle: "Time windows",
      iconSlug: "clock",
      values: [
        {
          badges: [
            {
              label: `${items.length} window${items.length > 1 ? "s" : ""}`,
              bgColor: "bg-gray-100",
              textColor: "text-gray-700",
            },
          ],
        },
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
