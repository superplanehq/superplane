import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "../mapperLookup";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { PrometheusQueryPayload, QueryRangeConfiguration, QueryRangeNodeMetadata } from "./types";

export const queryRangeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildQueryRangeProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    addQueryRangeOutputDetails(
      details,
      outputs.default[0].data as PrometheusQueryPayload,
      context.node?.configuration as QueryRangeConfiguration | undefined,
    );

    return details;
  },
};

function addQueryRangeOutputDetails(
  details: Record<string, string>,
  queryResult: PrometheusQueryPayload,
  configuration: QueryRangeConfiguration | undefined,
) {
  const fields: Array<[string, string | undefined]> = [
    ["Query", configuration?.query],
    ["Start", configuration?.start],
    ["End", configuration?.end],
    ["Step", configuration?.step],
    ["Result Type", queryResult?.resultType],
  ];

  for (const [label, value] of fields) {
    if (value) {
      details[label] = value;
    }
  }

  if (queryResult?.result !== undefined) {
    details["Results"] = String(Array.isArray(queryResult.result) ? queryResult.result.length : 0);
  }
}

function buildQueryRangeProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: { name: string; label: string; color: string },
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "unknown";

  return {
    iconSrc: prometheusIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    title: node.name || componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? buildEventSections(nodes, lastExecution, componentName) : undefined,
    metadata: getMetadata(node),
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as QueryRangeNodeMetadata | undefined;
  const configuration = node.configuration as QueryRangeConfiguration | undefined;

  const query = nodeMetadata?.query || configuration?.query;
  if (query) {
    metadata.push({ icon: "search", label: query });
  }

  if (configuration?.start) {
    metadata.push({ icon: "clock", label: `Start: ${configuration.start}` });
  }

  if (configuration?.end) {
    metadata.push({ icon: "clock", label: `End: ${configuration.end}` });
  }

  return metadata.slice(0, 3);
}

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
