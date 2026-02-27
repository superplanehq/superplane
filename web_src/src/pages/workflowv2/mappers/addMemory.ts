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

type AddMemoryMetadata = {
  namespace?: string;
  fields?: string[];
};

type AddMemoryConfiguration = {
  namespace?: string;
  valueList?: Array<{ name?: string; value?: unknown }>;
};

export const addMemoryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "addMemory";

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
      metadata: getAddMemoryMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = (context.node.metadata || {}) as AddMemoryMetadata;
    const namespace = (metadata.namespace || "").trim();
    const fields = Array.isArray(metadata.fields) ? metadata.fields.filter(Boolean) : [];

    if (namespace) {
      details["Namespace"] = namespace;
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

function getAddMemoryMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = (node.configuration || {}) as AddMemoryConfiguration;
  const metadata = (node.metadata || {}) as AddMemoryMetadata;
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

function extractConfiguredFields(config: AddMemoryConfiguration, metadata: AddMemoryMetadata): string[] {
  const configFields = Array.isArray(config.valueList)
    ? config.valueList.map((item) => (item?.name || "").trim()).filter((name): name is string => name.length > 0)
    : [];

  if (configFields.length > 0) {
    return Array.from(new Set(configFields));
  }

  return Array.isArray(metadata.fields) ? metadata.fields.filter(Boolean) : [];
}
