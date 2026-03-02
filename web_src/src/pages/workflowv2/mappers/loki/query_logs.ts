import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import lokiIcon from "@/assets/icons/integrations/loki.svg";
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
import { QueryLogsConfiguration, QueryLogsNodeMetadata, QueryLogsPayload } from "./types";

export const queryLogsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || context.node.componentName || "unknown";

    return {
      iconSrc: lokiIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: getMetadata(context.node),
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
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const queryResult = outputs.default[0].data as QueryLogsPayload;

    const configuration = context.node?.configuration as QueryLogsConfiguration | undefined;
    if (configuration?.query) {
      details["Query"] = configuration.query;
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

function getMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as QueryLogsNodeMetadata | undefined;
  const configuration = node.configuration as QueryLogsConfiguration | undefined;

  const query = nodeMetadata?.query || configuration?.query;
  if (query) {
    metadata.push({ icon: "search", label: query });
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
