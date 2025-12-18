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
import { MetadataItem } from "@/ui/metadataList";

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

    return {
      iconSlug: componentDefinition.icon || "circle-off",
      iconColor: "text-black",
      headerColor: "bg-white",
      metadata: getWaitMetadataList(node),
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecution
        ? getWaitEventSections(nodes, lastExecution, nodeQueueItems, node.configuration, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      hideMetadataList: false,
      eventStateMap: getStateMap(componentName),
    };
  },
};

function getWaitMetadataList(node: ComponentsNode): MetadataItem[] {
  const configuration = node.configuration as Record<string, unknown>;

  // Handle new mode-based configuration
  if (configuration?.mode) {
    const mode = configuration.mode as string;
    let waitText = "Wait";

    if (mode === "interval") {
      const waitFor = configuration.waitFor as string;
      const unit = configuration.unit as string;
      waitText = `Wait for ${waitFor}${unit ? ` ${unit}` : ""}`;
    } else if (mode === "countdown") {
      const waitUntil = configuration.waitUntil as string;
      waitText = `Wait until ${waitUntil}`;
    }

    return [
      {
        icon: "loader",
        label: waitText,
      },
    ];
  }

  // Handle legacy duration configuration for backward compatibility
  const duration = configuration?.duration as { value: number; unit: "seconds" | "minutes" | "hours" };
  if (duration) {
    return [
      {
        icon: "loader",
        label: `Wait for ${duration.value} ${duration.unit}`,
      },
    ];
  }

  // Fallback for no configuration
  return [
    {
      icon: "loader",
      label: "Wait configured",
    },
  ];
}

function getWaitEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[] | undefined,
  configuration: Record<string, unknown> | undefined,
  componentName: string,
): EventSection[] {
  const executionState = getState(componentName)(execution);
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  let eventSubtitle: string | React.ReactNode | undefined;

  // Calculate expected duration based on mode
  let expectedDuration: number | undefined;

  if (configuration?.mode === "interval") {
    const waitFor = configuration.waitFor as string;
    const unit = configuration.unit as string;

    // Try to parse waitFor as a number (for simple cases)
    const value = parseInt(waitFor, 10);
    if (!isNaN(value) && unit) {
      const multipliers = { seconds: 1000, minutes: 60000, hours: 3600000 };
      expectedDuration = value * (multipliers[unit as keyof typeof multipliers] || 1000);
    }
  } else if (configuration?.mode === "countdown") {
    const waitUntil = configuration.waitUntil as string;

    // Try to parse countdown target date
    if (waitUntil && execution.createdAt) {
      try {
        // For simple string dates, extract the date without evaluating expressions
        const dateMatch = waitUntil.match(/["']([^"']+)["']/);
        if (dateMatch) {
          const targetDate = new Date(dateMatch[1]);
          const createdDate = new Date(execution.createdAt);
          if (!isNaN(targetDate.getTime()) && !isNaN(createdDate.getTime())) {
            expectedDuration = targetDate.getTime() - createdDate.getTime();
          }
        }
      } catch (error) {
        // If parsing fails, expectedDuration remains undefined
      }
    }
  } else if (configuration?.duration) {
    // Legacy duration format
    const duration = configuration.duration as { value: number; unit: "seconds" | "minutes" | "hours" };
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

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle,
    eventState: executionState,
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}
