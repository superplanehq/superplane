import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { EventInfo, TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerProps } from "@/ui/trigger";
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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
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
    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: teamsIcon,
      iconSlug: "teams",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadata(
        node.metadata as OnMessageMetadata | undefined,
        node.configuration as OnMessageConfiguration | undefined,
      ),
    };

    if (lastEvent) {
      props.lastEventData = lastMessageEventData(lastEvent);
    }

    return props;
  },
};

function buildMetadata(
  metadata: OnMessageMetadata | undefined,
  configuration: OnMessageConfiguration | undefined,
): TriggerProps["metadata"] {
  const metadataItems: TriggerProps["metadata"] = [];
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

  return metadataItems;
}

function lastMessageEventData(lastEvent: NonNullable<EventInfo>): NonNullable<TriggerProps["lastEventData"]> {
  const eventData = lastEvent.data as MessageEventData | undefined;
  const title = eventData?.text?.trim() ? eventData.text : "Channel message";
  const subtitle = buildSubtitle(eventData?.from?.name || eventData?.from?.id || "", lastEvent.createdAt);

  return {
    title,
    subtitle,
    receivedAt: new Date(lastEvent.createdAt),
    state: "triggered",
    eventId: lastEvent.id,
  };
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

function buildSubtitle(content: string, createdAt?: string): string | React.ReactNode {
  if (content && createdAt) {
    return renderWithTimeAgo(content, new Date(createdAt));
  }
  return content || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}
