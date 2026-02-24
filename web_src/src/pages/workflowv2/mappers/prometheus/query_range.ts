import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { PrometheusQueryPayload, QueryRangeConfiguration, QueryRangeNodeMetadata } from "./types";

export const queryRangeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return buildQueryRangeProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }

    return formatTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const queryResult = outputs.default[0].data as PrometheusQueryPayload;

    const configuration = context.node?.configuration as QueryRangeConfiguration | undefined;
    if (configuration?.query) {
      details["Query"] = configuration.query;
    }

    if (configuration?.start) {
      details["Start"] = configuration.start;
    }

    if (configuration?.end) {
      details["End"] = configuration.end;
    }

    if (configuration?.step) {
      details["Step"] = configuration.step;
    }

    if (queryResult?.resultType) {
      details["Result Type"] = queryResult.resultType;
    }

    if (queryResult?.result !== undefined) {
      details["Results"] = String(Array.isArray(queryResult.result) ? queryResult.result.length : 0);
    }

    return details;
  },
};

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
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
