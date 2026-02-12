import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import { Tag } from "./types";
import { formatBytes, stringOrDash } from "../utils";

interface GetImageTagConfiguration {
  repository?: string;
  tag?: string;
}

export const getImageTagMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: dockerIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? getImageTagEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getImageTagMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Tag | undefined;

    if (!result) {
      return {};
    }

    const images = result.images;
    const image = Array.isArray(images) ? images[0] : images;

    return {
      Tag: stringOrDash(result.name),
      Status: stringOrDash(result.status),
      "Image Size": formatBytes(image?.size),
      "Full Size": formatBytes(result.full_size),
      "Last Updated": result.last_updated ? formatTimestampInUserTimezone(result.last_updated) : "-",
      "Last Pushed": result.tag_last_pushed ? formatTimestampInUserTimezone(result.tag_last_pushed) : "-",
      "Last Pulled": result.tag_last_pulled ? formatTimestampInUserTimezone(result.tag_last_pulled) : "-",
      "Last Updater": stringOrDash(result.last_updater_username),
      "Image Digest": stringOrDash(image?.digest),
      Architecture: stringOrDash(image?.architecture),
      OS: stringOrDash(image?.os),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getImageTagMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as GetImageTagConfiguration | undefined;

  if (configuration?.repository) {
    metadata.push({ icon: "package", label: configuration.repository });
  }

  if (configuration?.tag) {
    metadata.push({ icon: "tag", label: configuration.tag });
  }

  return metadata;
}

function getImageTagEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
