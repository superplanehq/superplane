import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
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
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "dockerhub";

    return {
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Docker Hub",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default: OutputPayload[] };
    if (!outputs?.default?.[0]?.data) {
      return {};
    }
    const data = outputs.default[0].data as ListTagsResponse;
    return getDetailsForResponse(data);
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
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

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
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
