import {
  ComponentsNode,
  ComponentsComponent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
} from "@/api-client";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { ListTagsResponse, Tag } from "./types";
import { formatTimeAgo } from "@/utils/date";

interface ListTagsConfiguration {
  repository?: string;
  pageSize?: number;
  nameFilter?: string;
}

export const listTagsMapper: ComponentBaseMapper = {
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
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: CanvasesCanvasNodeExecution, _: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default: OutputPayload[] };
    if (!outputs?.default?.[0]?.data) {
      return {};
    }
    const data = outputs.default[0].data as ListTagsResponse;
    return getDetailsForResponse(data);
  },

  subtitle(_node: ComponentsNode, execution: CanvasesCanvasNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as ListTagsConfiguration;

  if (configuration.repository) {
    metadata.push({ icon: "box", label: configuration.repository });
  }

  if (configuration.nameFilter) {
    metadata.push({ icon: "filter", label: `Filter: ${configuration.nameFilter}` });
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
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDetailsForResponse(data: ListTagsResponse): Record<string, string> {
  const details: Record<string, string> = {};

  if (data?.count !== undefined) {
    details["Total Tags"] = String(data.count);
  }

  if (data?.results && data.results.length > 0) {
    details["Tags Retrieved"] = String(data.results.length);

    const firstTag = data.results[0] as Tag;
    if (firstTag?.name) {
      details["First Tag"] = firstTag.name;
    }
    if (firstTag?.last_updated) {
      details["First Tag Updated"] = new Date(firstTag.last_updated).toLocaleString();
    }
  }

  return details;
}
