import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type React from "react";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../types";

/** When set, matches the historical Query Data Source node event strip (subtitle + id fallbacks). */
export type BuildGrafanaEventSectionsOptions = {
  legacyQueryDataSource?: boolean;
  /** When true, return [] if root trigger event or timestamp is missing (alert rule mappers). */
  strict?: boolean;
};

export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  options?: BuildGrafanaEventSectionsOptions,
): EventSection[] {
  const legacy = options?.legacyQueryDataSource === true;
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
      receivedAt: resolveGrafanaEventReceivedAt(execution, legacy),
      eventTitle: title || "Trigger event",
      eventSubtitle: resolveGrafanaEventSubtitle(execution, legacy),
      eventState: getState(componentName)(execution),
      eventId: resolveGrafanaEventId(execution, legacy),
    },
  ];
}

function resolveGrafanaEventSubtitle(execution: ExecutionInfo, legacy: boolean): string | React.ReactNode {
  const timestamp = legacy ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const empty = legacy ? "-" : "";
  return timestamp ? renderTimeAgo(new Date(timestamp)) : empty;
}

function resolveGrafanaEventReceivedAt(execution: ExecutionInfo, legacy: boolean): Date | undefined {
  const raw = legacy ? execution.createdAt : execution.createdAt || execution.updatedAt;
  return raw ? new Date(raw) : undefined;
}

function resolveGrafanaEventId(execution: ExecutionInfo, legacy: boolean): string {
  if (legacy) return execution.rootEvent?.id ?? "";
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
