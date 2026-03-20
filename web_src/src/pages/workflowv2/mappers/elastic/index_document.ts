import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { formatTimeAgo } from "@/utils/date";

interface IndexDocumentConfiguration {
  index?: string;
  documentId?: string;
}

type UnknownRecord = Record<string, unknown>;

interface OutputPayloadView {
  timestamp?: string;
  data?: unknown;
}

interface IndexDocumentOutputData {
  id?: string;
  index?: string;
  result?: string;
  version?: string | number;
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
    const doc = toIndexDocumentOutputData(payload?.data);
    if (!doc) {
      return {};
    }
    return getDetailsForDocument(doc, payload?.timestamp);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = toIndexDocumentConfiguration(node.configuration);

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

function getDetailsForDocument(doc: IndexDocumentOutputData, timestamp?: string): Record<string, string> {
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

function getFirstOutputPayload(outputs: unknown): OutputPayloadView | undefined {
  const outputRecord = toUnknownRecord(outputs);
  if (!outputRecord) return undefined;

  const payload =
    getOutputPayloadFromChannel(outputRecord.default) ||
    getOutputPayloadFromChannel(outputRecord.success) ||
    getOutputPayloadFromChannel(outputRecord.failed);

  if (!payload) return undefined;

  return {
    timestamp: toOptionalString(payload.timestamp),
    data: payload.data,
  };
}

function getOutputPayloadFromChannel(channel: unknown): UnknownRecord | undefined {
  if (!Array.isArray(channel) || channel.length === 0) return undefined;
  return toUnknownRecord(channel[0]);
}

function toIndexDocumentConfiguration(value: unknown): IndexDocumentConfiguration {
  const config = toUnknownRecord(value);
  return {
    index: toOptionalString(config?.index),
    documentId: toOptionalString(config?.documentId),
  };
}

function toIndexDocumentOutputData(value: unknown): IndexDocumentOutputData | undefined {
  const payload = toUnknownRecord(value);
  if (!payload) return undefined;

  const version = payload.version;
  const parsedVersion = typeof version === "number" || typeof version === "string" ? version : undefined;

  return {
    id: toOptionalString(payload.id),
    index: toOptionalString(payload.index),
    result: toOptionalString(payload.result),
    version: parsedVersion,
  };
}

function toUnknownRecord(value: unknown): UnknownRecord | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  return value as UnknownRecord;
}

function toOptionalString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}
