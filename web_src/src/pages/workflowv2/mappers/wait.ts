import React from "react";
import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { TimeLeftCountdown } from "@/ui/timeLeftCountdown";
import { formatTimestamp } from "@/lib/utils";

export const waitMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    nodeQueueItems?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "wait";
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const duration = node.configuration?.duration as { value: number; unit: "seconds" | "minutes" | "hours" };

    const metadata = [
      {
        icon: "clock",
        label: duration ? `${duration.value} ${duration.unit}` : "No duration configured",
      },
    ];

    return {
      iconSlug: componentDefinition.icon || "circle-off",
      iconColor: "text-black",
      headerColor: "bg-white",
      metadata,
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: getWaitEventSections(nodes, lastExecution, nodeQueueItems, duration, componentName),
      hideMetadataList: true,
      eventStateMap: getStateMap(componentName),
    };
  },
};

function getWaitEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution | null,
  nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
  duration: { value: number; unit: "seconds" | "minutes" | "hours" },
  componentName: string,
): EventSection[] {
  const sections: EventSection[] = [];

  // Add Last Event section
  if (!execution) {
    sections.push({
      eventTitle: "No events received yet",
      eventState: "neutral" as const,
    });
  } else {
    const executionState = getState(componentName)(execution);
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    let eventSubtitle: string | React.ReactNode | undefined;

    let expectedDuration: number | undefined;
    if (duration) {
      const { value, unit } = duration;
      const multipliers = { seconds: 1000, minutes: 60000, hours: 3600000 };
      expectedDuration = value * (multipliers[unit as keyof typeof multipliers] || 1000);
    }

    if (executionState === "running" && execution.createdAt && expectedDuration) {
      eventSubtitle = React.createElement(TimeLeftCountdown, {
        createdAt: new Date(execution.createdAt),
        expectedDuration: expectedDuration,
      });
    }

    if (executionState === "success" || executionState === "failed") {
      if (execution.updatedAt) {
        eventSubtitle = `Done at: ${formatTimestamp(new Date(execution.updatedAt))} `;
      } else {
        eventSubtitle = "Done";
      }
    }

    sections.push({
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: executionState,
    });
  }

  if (nodeQueueItems && nodeQueueItems.length > 0) {
    const queueItem = nodeQueueItems[nodeQueueItems.length - 1];
    const rootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    if (queueItem.rootEvent) {
      const { title } = rootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent);
      sections.push({
        receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : undefined,
        eventTitle: title,
        eventState: "next-in-queue" as const,
      });
    }
  }

  return sections;
}
