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
import { Tag } from "./types";
import { formatTimeAgo } from "@/utils/date";

interface DescribeImageTagConfiguration {
  namespace?: string;
  repository?: string;
  tag?: string;
}

export const describeImageTagMapper: ComponentBaseMapper = {
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
    const data = outputs.default[0].data as Tag;
    return getDetailsForTag(data);
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DescribeImageTagConfiguration;

  if (configuration.namespace && configuration.repository) {
    metadata.push({ icon: "box", label: `${configuration.namespace}/${configuration.repository}` });
  }

  if (configuration.tag) {
    metadata.push({ icon: "tag", label: configuration.tag });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  let eventId: string | undefined = undefined;
  let title = "";
  if (rootEvent) {
    eventId = rootEvent.id;
    const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
    title = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent }).title;
  }
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}

function getDetailsForTag(data: Tag): Record<string, string> {
  const details: Record<string, string> = {};

  if (data?.name) {
    details["Tag Name"] = data.name;
  }

  if (data?.digest) {
    details["Digest"] = data.digest.substring(0, 20) + "...";
  }

  if (data?.full_size !== undefined) {
    // Convert to human readable size
    const sizeInMB = (data.full_size / (1024 * 1024)).toFixed(2);
    details["Size"] = `${sizeInMB} MB`;
  }

  if (data?.last_updated) {
    details["Last Updated"] = formatTimeAgo(new Date(data.last_updated));
  }

  if (data?.images && data.images.length > 0) {
    const platforms = data.images.map((img) => `${img.os}/${img.architecture}`).join(", ");
    details["Platforms"] = platforms;
  }

  return details;
}
