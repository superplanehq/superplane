import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import {
  ComponentBaseProps,
  DEFAULT_EVENT_STATE_MAP,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { formatTimeAgo } from "@/utils/date";
import { defaultStateFunction } from "./stateRegistry";

type DeleteMemoryMetadata = {
  namespace?: string;
  fields?: string[];
  matches?: Record<string, unknown>;
};

type DeleteMemoryConfiguration = {
  namespace?: string;
  matchList?: Array<{ name?: string; value?: unknown }>;
};

type DeleteMemoryOutputs = {
  deleted?: OutputPayload[];
  notFound?: OutputPayload[];
  default?: OutputPayload[]; // Backwards compatibility for legacy executions.
};

const DELETE_MEMORY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  notFound: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "Not Found",
  },
};

function getDeleteMemoryState(execution: ExecutionInfo): EventState {
  const defaultState = defaultStateFunction(execution);
  if (defaultState !== "success") {
    return defaultState;
  }

  const outputs = execution.outputs as DeleteMemoryOutputs | undefined;
  if (outputs?.notFound && outputs.notFound.length > 0) {
    return "notFound";
  }

  return "success";
}

export const deleteMemoryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

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
      metadata: getDeleteMemoryMetadataList(context.node),
      eventStateMap: DELETE_MEMORY_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = (context.node.metadata || {}) as DeleteMemoryMetadata;
    const namespace = (metadata.namespace || "").trim();
    const fields = extractConfiguredFields((context.node.configuration || {}) as DeleteMemoryConfiguration, metadata);

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
      eventState: getDeleteMemoryState(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getDeleteMemoryMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = (node.configuration || {}) as DeleteMemoryConfiguration;
  const metadata = (node.metadata || {}) as DeleteMemoryMetadata;
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

function extractConfiguredFields(config: DeleteMemoryConfiguration, metadata: DeleteMemoryMetadata): string[] {
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
