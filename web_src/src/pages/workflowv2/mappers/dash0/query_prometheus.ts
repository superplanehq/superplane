import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { QueryPrometheusConfiguration } from "./types";
import { formatTimeAgo } from "@/utils/date";

export const queryPrometheusMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: CanvasesCanvasNodeExecution[],
    _?: CanvasesCanvasNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.component?.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

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
  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as unknown as QueryPrometheusConfiguration;

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
  nodes: ComponentsNode[],
  execution: CanvasesCanvasNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

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
