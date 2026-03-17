import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, DEFAULT_EVENT_STATE_MAP, EventSection, EventStateMap } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import discordIcon from "@/assets/icons/integrations/discord.svg";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { defaultStateFunction } from "../stateRegistry";

interface GetLastMentionConfiguration {
  channel?: string;
  since?: string;
}

interface ChannelMetadata {
  id?: string;
  name?: string;
}

interface GetLastMentionMetadata {
  channel?: ChannelMetadata;
}

interface MentionPayload {
  channel_id?: string;
  mention?: {
    id?: string;
    content?: string;
    timestamp?: string;
    author?: {
      id?: string;
      username?: string;
    };
  };
}

const GET_LAST_MENTION_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  found: {
    icon: "at-sign",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
    label: "Found",
  },
  notFound: {
    icon: "message-circle-off",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
    label: "Not Found",
  },
};

const getLastMentionStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const defaultState = defaultStateFunction(execution);
  if (defaultState !== "success") {
    return defaultState;
  }

  const outputs = execution.outputs as
    | { found?: OutputPayload[]; notFound?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;

  if (outputs?.found?.length) {
    return "found";
  }

  if (outputs?.notFound?.length) {
    return "notFound";
  }

  const data = outputs?.default?.[0]?.data as MentionPayload | undefined;
  return data?.mention ? "found" : "notFound";
};

export const GET_LAST_MENTION_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_LAST_MENTION_STATE_MAP,
  getState: getLastMentionStateFunction,
};

export const getLastMentionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Get Last Mention",
      iconSrc: discordIcon,
      iconSlug: "discord",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? getLastMentionEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: getLastMentionMetadataList(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as
      | { found?: OutputPayload[]; notFound?: OutputPayload[]; default?: OutputPayload[] }
      | undefined;
    const data =
      (outputs?.found?.[0]?.data as MentionPayload | undefined) ??
      (outputs?.notFound?.[0]?.data as MentionPayload | undefined) ??
      (outputs?.default?.[0]?.data as MentionPayload | undefined);
    const found = Boolean(outputs?.found?.length || data?.mention?.id);

    return {
      "Channel ID": stringOrDash(data?.channel_id),
      Result: found ? "Found" : "Not Found",
      "Mention ID": stringOrDash(data?.mention?.id),
      Author: data?.mention?.author?.username
        ? `@${data.mention.author.username}`
        : stringOrDash(data?.mention?.author?.id),
      Content: stringOrDash(data?.mention?.content),
      Timestamp: stringOrDash(data?.mention?.timestamp),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getLastMentionMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GetLastMentionMetadata | undefined;
  const configuration = node.configuration as GetLastMentionConfiguration | undefined;

  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  return metadata;
}

function getLastMentionEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
