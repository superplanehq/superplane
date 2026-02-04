import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, ExecutionDetailsContext, ComponentBaseContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { QueryPrometheusConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const queryPrometheusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return { Response: "No data returned" };
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, any> | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    // Format the Prometheus response data for display
    const details: Record<string, string> = {};
    if (payload?.timestamp) {
      details["Checked At"] = new Date(payload.timestamp).toLocaleString();
    }
    try {
      const formatted = JSON.stringify(responseData, null, 2);
      details["Response Data"] = formatted;
    } catch (error) {
      details["Response Data"] = String(responseData);
    }

    return details;
  },
  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryPrometheusConfiguration;

  if (configuration?.query) {
    // Show a preview of the query (first 50 chars)
    const queryPreview =
      configuration.query.length > 50 ? configuration.query.substring(0, 50) + "..." : configuration.query;
    metadata.push({ icon: "code", label: queryPreview });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: `Dataset: ${configuration.dataset}` });
  }

  if (configuration?.type) {
    metadata.push({ icon: "funnel", label: `Type: ${configuration.type}` });
  }

  return metadata;
}

function baseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
