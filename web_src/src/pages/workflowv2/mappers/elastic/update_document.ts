import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
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
import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

interface UpdateDocumentConfiguration {
  index?: string;
  document?: string;
}

interface UpdateDocumentOutputData {
  id?: string;
  index?: string;
  result?: string;
  version?: number;
}

export const updateDocumentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: elasticIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default: OutputPayload[] };
    const details: Record<string, string> = {};
    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }
    if (!outputs?.default?.[0]?.data) {
      return details;
    }
    const doc = outputs.default[0].data as UpdateDocumentOutputData;
    return { ...details, ...getDetailsForUpdateDocument(doc) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateDocumentConfiguration | undefined;

  if (configuration?.index) {
    metadata.push({ icon: "database", label: configuration.index });
  }

  if (configuration?.document) {
    metadata.push({ icon: "hash", label: configuration.document });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const triggerComponent = rootTriggerNode?.componentName ?? componentName;
  const rootTriggerRenderer = getTriggerRenderer(triggerComponent);
  const titleAndSubtitle = rootTriggerRenderer?.getTitleAndSubtitle({ event: execution.rootEvent });
  const title = titleAndSubtitle?.title ?? "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDetailsForUpdateDocument(doc: UpdateDocumentOutputData): Record<string, string> {
  const details: Record<string, string> = {};

  if (doc?.id) {
    details["Document"] = String(doc.id);
  }

  if (doc?.index) {
    details["Index"] = doc.index;
  }

  if (doc?.result) {
    details["Result"] = doc.result;
  }

  if (doc?.version != null) {
    details["Version"] = String(doc.version);
  }

  return details;
}
