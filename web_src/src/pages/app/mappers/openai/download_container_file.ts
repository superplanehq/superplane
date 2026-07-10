import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatBytes } from "./base";

type DownloadContainerFileConfiguration = {
  containerId?: string;
  fileId?: string;
};

type ContainerFilePayloadData = {
  fileId?: string;
  containerId?: string;
  path?: string;
  filename?: string;
  bytes?: number;
  encoding?: string;
  content?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

// Container and file ids are typically expressions resolved at run time (e.g.
// from a Text Prompt artifact), so templated values are not shown on the card.
function isLiteral(value: string | undefined): value is string {
  return Boolean(value && !value.includes("{{"));
}

export const downloadContainerFileMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openai";

    return {
      iconSrc: openAiIcon,
      iconSlug: context.componentDefinition?.icon ?? "file-down",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Download Container File",
      eventSections: lastExecution
        ? downloadContainerFileEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    if (timestamp) {
      details["Executed At"] = new Date(timestamp).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = (outputs?.default?.[0]?.data ?? {}) as ContainerFilePayloadData;

    addDetail(details, "Filename", data.filename);
    addDetail(details, "Path", data.path);
    addDetail(details, "Size", formatBytes(data.bytes));
    addDetail(details, "Container ID", data.containerId);
    addDetail(details, "File ID", data.fileId);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

// metadataList shows the configured container and file ids on the component
// card, skipping values that are expressions (they only resolve at run time).
function metadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration ?? {}) as DownloadContainerFileConfiguration;
  const items: MetadataItem[] = [];

  if (isLiteral(config.containerId)) {
    items.push({ icon: "container", label: config.containerId });
  }
  if (isLiteral(config.fileId)) {
    items.push({ icon: "file-text", label: config.fileId });
  }

  return items;
}

function downloadContainerFileEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

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
