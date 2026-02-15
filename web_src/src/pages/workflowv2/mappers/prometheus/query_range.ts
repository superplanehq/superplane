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
import { QueryRangeConfiguration, PrometheusQueryResultPayload } from "./types";

export const queryRangeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { nodes, node, componentDefinition, lastExecutions } = context;
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

    const result = outputs.default[0].data as PrometheusQueryResultPayload;

    if (result?.query) {
      details["Query"] = result.query;
    }

    if (result?.start) {
      details["Start"] = result.start;
    }

    if (result?.end) {
      details["End"] = result.end;
    }

    if (result?.step) {
      details["Step"] = result.step;
    }

    if (result?.resultType) {
      details["Result Type"] = result.resultType;
    }

    if (result?.result) {
      details["Result Count"] = `${result.result.length} series`;
    }

    return details;
  },
};

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryRangeConfiguration | undefined;

  if (configuration?.query) {
    const truncatedQuery =
      configuration.query.length > 30 ? configuration.query.substring(0, 30) + "…" : configuration.query;
    metadata.push({ icon: "search", label: truncatedQuery });
  }

  if (configuration?.step) {
    metadata.push({ icon: "clock", label: `Step: ${configuration.step}` });
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
