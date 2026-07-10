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
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatBytes } from "./base";

type FileNodeMetadata = {
  filename?: string;
};

type DownloadFileConfiguration = {
  file?: string;
};

type DownloadedFilePayloadData = {
  id?: string;
  filename?: string;
  mimeType?: string;
  sizeBytes?: number;
  encoding?: string;
  content?: string;
};

function addDetail(details: Record<string, string>, key: string, value: string | undefined) {
  if (value) {
    details[key] = value;
  }
}

export const downloadFileMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "file-down",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "Download File",
      eventSections: lastExecution ? downloadFileEventSections(context.nodes, lastExecution, componentName) : undefined,
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
    const data = (outputs?.default?.[0]?.data ?? {}) as DownloadedFilePayloadData;

    addDetail(details, "Filename", data.filename);
    addDetail(details, "MIME Type", data.mimeType);
    addDetail(details, "Size", formatBytes(data.sizeBytes));
    addDetail(details, "Encoding", data.encoding);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

// metadataList shows the configured file on the component card. The filename
// prefers backend node metadata (resolved in Setup) and falls back to the raw
// configuration value so it shows before the first save round-trip.
function metadataList(node: NodeInfo): MetadataItem[] {
  const meta = (node.metadata ?? {}) as FileNodeMetadata;
  const config = (node.configuration ?? {}) as DownloadFileConfiguration;

  const file = meta.filename || config.file;
  return file ? [{ icon: "file-text", label: file }] : [];
}

function downloadFileEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
