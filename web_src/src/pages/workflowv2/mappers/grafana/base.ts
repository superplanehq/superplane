import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../types";

export type BuildGrafanaEventSectionsOptions = {
  /** When true, return [] if root trigger event or timestamp is missing (alert rule mappers). */
  strict?: boolean;
};

export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  options?: BuildGrafanaEventSectionsOptions,
): EventSection[] {
  const strict = options?.strict === true;

  if (strict) {
    if (!execution.rootEvent?.id || !execution.createdAt) {
      return [];
    }
    const strictTrigger = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
    if (!strictTrigger?.componentName) {
      return [];
    }
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const triggerName = rootTriggerNode?.componentName ?? "";
  const rootTriggerRenderer = getTriggerRenderer(triggerName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: resolveGrafanaEventReceivedAt(execution),
      eventTitle: title || "Trigger event",
      eventSubtitle: resolveGrafanaEventSubtitle(execution),
      eventState: getState(componentName)(execution),
      eventId: resolveGrafanaEventId(execution),
    },
  ];
}

/** Single source for Grafana event display time so subtitle and receivedAt stay aligned. */
function resolveGrafanaEventDisplayTimestamp(execution: ExecutionInfo): string | undefined {
  return execution.createdAt || execution.updatedAt;
}

function resolveGrafanaEventSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const timestamp = resolveGrafanaEventDisplayTimestamp(execution);
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function resolveGrafanaEventReceivedAt(execution: ExecutionInfo): Date | undefined {
  const raw = resolveGrafanaEventDisplayTimestamp(execution);
  return raw ? new Date(raw) : undefined;
}

function resolveGrafanaEventId(execution: ExecutionInfo): string {
  return execution.rootEvent?.id ?? execution.id;
}

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
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const eventTitle = title || "Trigger event";

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle: eventTitle,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "-",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
