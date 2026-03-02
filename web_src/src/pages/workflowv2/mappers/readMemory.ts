import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "./types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getStateMap, getTriggerRenderer } from ".";
import { formatTimeAgo } from "@/utils/date";
import { defaultStateFunction } from "./stateRegistry";

type ReadMemoryMetadata = {
  namespace?: string;
  fields?: string[];
  matches?: Record<string, unknown>;
  resultMode?: string;
  emitMode?: string;
};

type ReadMemoryConfiguration = {
  namespace?: string;
  resultMode?: string;
  emitMode?: string;
  matchList?: Array<{ name?: string; value?: unknown }>;
};

export const readMemoryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "readMemory";

    return {
      iconSlug: context.componentDefinition.icon ?? "database",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: getReadMemoryMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = (context.node.metadata || {}) as ReadMemoryMetadata;
    const namespace = (metadata.namespace || "").trim();
    const resultMode = extractResultMode((context.node.configuration || {}) as ReadMemoryConfiguration, metadata);
    const emitMode = extractEmitMode((context.node.configuration || {}) as ReadMemoryConfiguration, metadata);
    const fields = extractConfiguredFields((context.node.configuration || {}) as ReadMemoryConfiguration, metadata);

    if (namespace) {
      details["Namespace"] = namespace;
    }
    if (resultMode) {
      details["Result Mode"] = resultMode === "latest" ? "Latest Match" : "All Matches";
    }
    if (emitMode) {
      details["Emit Mode"] = emitMode === "oneByOne" ? "One By One" : "All At Once";
    }
    if (fields.length > 0) {
      details["Fields"] = fields.join(", ");
    }

    return details;
  },
};

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title: fallbackTitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: fallbackTitle,
      eventSubtitle,
      eventState: defaultStateFunction(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getReadMemoryMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = (node.configuration || {}) as ReadMemoryConfiguration;
  const metadata = (node.metadata || {}) as ReadMemoryMetadata;
  const namespace = ((config.namespace as string) || metadata.namespace || "").trim();
  const fields = extractConfiguredFields(config, metadata);
  const items: Array<{ icon: string; label: string }> = [];

  if (namespace) {
    items.push({ icon: "database", label: namespace });
  }
  if (fields.length > 0) {
    items.push({ icon: "list", label: fields.join(", ") });
  }

  return items;
}

function extractConfiguredFields(config: ReadMemoryConfiguration, metadata: ReadMemoryMetadata): string[] {
  const configFields = Array.isArray(config.matchList)
    ? config.matchList.map((item) => (item?.name || "").trim()).filter((name): name is string => name.length > 0)
    : [];

  if (configFields.length > 0) {
    return Array.from(new Set(configFields));
  }

  if (Array.isArray(metadata.fields) && metadata.fields.length > 0) {
    return metadata.fields.filter(Boolean);
  }

  return metadata.matches ? Object.keys(metadata.matches).filter((key) => key.trim().length > 0) : [];
}

function extractResultMode(config: ReadMemoryConfiguration, metadata: ReadMemoryMetadata): string {
  const value = ((config.resultMode as string) || metadata.resultMode || "").trim().toLowerCase();
  if (value === "latest") {
    return "latest";
  }
  return value === "all" ? "all" : "";
}

function extractEmitMode(config: ReadMemoryConfiguration, metadata: ReadMemoryMetadata): string {
  const resultMode = extractResultMode(config, metadata);
  if (resultMode !== "all") {
    return "";
  }
  const value = ((config.emitMode as string) || metadata.emitMode || "").trim();
  if (value === "oneByOne") {
    return "oneByOne";
  }
  return value === "allAtOnce" ? "allAtOnce" : "";
}
