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

type UpdateMemoryMetadata = {
  namespace?: string;
  matchFields?: string[];
  valueFields?: string[];
  matches?: Record<string, unknown>;
  updatedCount?: number;
};

type UpdateMemoryConfiguration = {
  namespace?: string;
  matchList?: Array<{ name?: string; value?: unknown }>;
  valueList?: Array<{ name?: string; value?: unknown }>;
};

type UpdateMemoryOutputs = {
  found?: OutputPayload[];
  notFound?: OutputPayload[];
  default?: OutputPayload[];
};

const UPDATE_MEMORY_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  notFound: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "Not Found",
  },
};

function getUpdateMemoryState(execution: ExecutionInfo): EventState {
  const defaultState = defaultStateFunction(execution);
  if (defaultState !== "success") {
    return defaultState;
  }

  const outputs = execution.outputs as UpdateMemoryOutputs | undefined;
  if (outputs?.notFound && outputs.notFound.length > 0) {
    return "notFound";
  }

  return "success";
}

export const updateMemoryMapper: ComponentBaseMapper = {
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
      metadata: getUpdateMemoryMetadataList(context.node),
      eventStateMap: UPDATE_MEMORY_STATE_MAP,
    };
  },
  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const metadata = (context.node.metadata || {}) as UpdateMemoryMetadata;
    const namespace = (metadata.namespace || "").trim();
    const matchFields = extractConfiguredFields(
      ((context.node.configuration || {}) as UpdateMemoryConfiguration).matchList,
      metadata.matchFields,
      metadata.matches,
    );
    const valueFields = extractConfiguredFields(
      ((context.node.configuration || {}) as UpdateMemoryConfiguration).valueList,
      metadata.valueFields,
      undefined,
    );

    if (namespace) {
      details["Namespace"] = namespace;
    }
    if (matchFields.length > 0) {
      details["Match Fields"] = matchFields.join(", ");
    }
    if (valueFields.length > 0) {
      details["Updated Fields"] = valueFields.join(", ");
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
      eventState: getUpdateMemoryState(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function getUpdateMemoryMetadataList(node: NodeInfo): Array<{ icon: string; label: string }> {
  const config = (node.configuration || {}) as UpdateMemoryConfiguration;
  const metadata = (node.metadata || {}) as UpdateMemoryMetadata;
  const namespace = ((config.namespace as string) || metadata.namespace || "").trim();
  const matchFields = extractConfiguredFields(config.matchList, metadata.matchFields, metadata.matches);
  const valueFields = extractConfiguredFields(config.valueList, metadata.valueFields, undefined);
  const items: Array<{ icon: string; label: string }> = [];

  if (namespace) {
    items.push({ icon: "database", label: namespace });
  }
  if (matchFields.length > 0) {
    items.push({ icon: "search", label: `match: ${matchFields.join(", ")}` });
  }
  if (valueFields.length > 0) {
    items.push({ icon: "list", label: `set: ${valueFields.join(", ")}` });
  }

  return items;
}

function extractConfiguredFields(
  list: Array<{ name?: string; value?: unknown }> | undefined,
  metadataFields: string[] | undefined,
  metadataMatches: Record<string, unknown> | undefined,
): string[] {
  const configFields = Array.isArray(list)
    ? list.map((item) => (item?.name || "").trim()).filter((name): name is string => name.length > 0)
    : [];

  if (configFields.length > 0) {
    return Array.from(new Set(configFields));
  }

  if (Array.isArray(metadataFields) && metadataFields.length > 0) {
    return metadataFields.filter(Boolean);
  }

  return metadataMatches ? Object.keys(metadataMatches).filter((key) => key.trim().length > 0) : [];
}
