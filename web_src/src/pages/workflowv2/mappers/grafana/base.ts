import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../types";

export function grafanaComponentBaseProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "unknown";

  return {
    iconSrc: grafanaIcon,
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

export function grafanaCreatedAtSubtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) return "-";
  return renderTimeAgo(new Date(context.execution.createdAt));
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  return grafanaEventSections(nodes, execution, componentName, "lenient");
}

/** Returns [] when root trigger metadata is missing (used by alert-rule mappers). */
export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  return grafanaEventSections(nodes, execution, componentName, "strict");
}

function grafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  mode: "strict" | "lenient",
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);

  if (mode === "strict") {
    if (!execution.rootEvent?.id || !execution.createdAt) {
      return [];
    }
    if (!rootTriggerNode?.componentName) {
      return [];
    }
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const eventTitle = title || "Trigger event";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "-",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
