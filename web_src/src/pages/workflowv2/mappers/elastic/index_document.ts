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
import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { formatTimeAgo } from "@/utils/date";

interface IndexDocumentConfiguration {
  index?: string;
  documentId?: string;
}

export const indexDocumentMapper: ComponentBaseMapper = {
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
    const payload = getFirstOutputPayload(context.execution.outputs);
    if (!payload?.data) {
      return {};
    }
    const doc = payload.data as Record<string, any>;

    return getDetailsForDocument(doc, payload.timestamp);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as IndexDocumentConfiguration | undefined;

  if (configuration?.index) {
    metadata.push({ icon: "database", label: configuration.index });
  }

  if (configuration?.documentId) {
    metadata.push({ icon: "hash", label: `ID: ${configuration.documentId}` });
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

function getFirstOutputPayload(outputs: unknown): OutputPayload | undefined {
  const typedOutputs = outputs as
    | { success?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;
  return typedOutputs?.default?.[0] ?? typedOutputs?.success?.[0] ?? typedOutputs?.failed?.[0];
}

function getDetailsForDocument(doc: Record<string, any>, timestamp?: string): Record<string, string> {
  const details: Record<string, string> = {};

  if (timestamp) {
    details["Indexed At"] = new Date(timestamp).toLocaleString();
  }

  if (doc?.id) {
    details["Document ID"] = String(doc.id);
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
