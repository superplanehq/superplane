import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { TriggerProps } from "@/ui/trigger";
import teamsIcon from "@/assets/icons/integrations/teams.svg";

interface OnMessageConfiguration {
  channel?: string;
  contentFilter?: string;
}

interface OnMessageMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

interface MessageEventData {
  text?: string;
  from?: {
    id?: string;
    name?: string;
  };
  conversation?: {
    id?: string;
    name?: string;
  };
  timestamp?: string;
}

/**
 * Renderer for the "teams.onMessage" trigger
 */
export const onMessageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as MessageEventData | undefined;
    const title = eventData?.text?.trim() ? eventData.text : "Channel message";
    const subtitle = buildSubtitle(eventData?.from?.name || eventData?.from?.id || "", context.event?.createdAt);

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as MessageEventData | undefined;

    return {
      Channel: stringOrDash(eventData?.conversation?.name || eventData?.conversation?.id),
      User: stringOrDash(eventData?.from?.name || eventData?.from?.id),
      Text: stringOrDash(eventData?.text),
      Timestamp: stringOrDash(eventData?.timestamp),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnMessageMetadata | undefined;
    const configuration = node.configuration as OnMessageConfiguration | undefined;
    const metadataItems = [];

    const channelLabel = metadata?.channel?.name || configuration?.channel;
    if (channelLabel) {
      metadataItems.push({
        icon: "hash",
        label: channelLabel,
      });
    }

    if (configuration?.contentFilter) {
      metadataItems.push({
        icon: "funnel",
        label: `Filter: ${configuration.contentFilter}`,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: teamsIcon,
      iconSlug: "teams",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as MessageEventData | undefined;
      const title = eventData?.text?.trim() ? eventData.text : "Channel message";
      const subtitle = buildSubtitle(eventData?.from?.name || eventData?.from?.id || "", lastEvent.createdAt);

      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}
