import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerProps } from "@/ui/trigger";
import slackIcon from "@/assets/icons/integrations/slack.svg";

interface OnReactionAddedConfiguration {
  channel?: string;
  reaction?: string;
}

interface OnReactionAddedMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

interface ReactionAddedEventData {
  type?: string;
  channel?: string;
  timestamp?: string;
  reaction?: string;
  item?: {
    type?: string;
    channel?: string;
    ts?: string;
  };
  user?: string;
}

/**
 * Renderer for the "slack.onReactionAdded" trigger
 */
export const onReactionAddedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as ReactionAddedEventData | undefined;
    const title = eventData?.reaction ? `${eventData.reaction} reaction` : "Reaction added";
    const subtitle = buildSubtitle(
      eventData?.user ? `Added by ${eventData.user}` : "Reaction",
      context.event?.createdAt,
    );

    return {
      title,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as ReactionAddedEventData | undefined;
    const reactionAt = formatSlackTimestamp(eventData?.timestamp);

    return {
      "Reaction At": reactionAt || "",
      Channel: stringOrDash(eventData?.item?.channel),
      Reaction: stringOrDash(eventData?.reaction),
      User: stringOrDash(eventData?.user),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnReactionAddedMetadata | undefined;
    const configuration = node.configuration as OnReactionAddedConfiguration | undefined;
    const metadataItems = [];

    const channelLabel = metadata?.channel?.name || configuration?.channel;
    if (channelLabel) {
      metadataItems.push({
        icon: "hash",
        label: channelLabel,
      });
    }

    if (configuration?.reaction) {
      metadataItems.push({
        icon: "smile",
        label: configuration.reaction,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as ReactionAddedEventData | undefined;
      const title = eventData?.reaction ? `${eventData.reaction} reaction` : "Reaction added";
      const subtitle = buildSubtitle(eventData?.user ? `Added by ${eventData.user}` : "Reaction", lastEvent.createdAt);

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

function buildSubtitle(content: string, createdAt?: string): string | React.ReactNode {
  if (content && createdAt) {
    return renderWithTimeAgo(content, new Date(createdAt));
  }
  return content || (createdAt ? renderTimeAgo(new Date(createdAt)) : "");
}

function formatSlackTimestamp(value?: unknown): string | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }

  const raw = String(value);
  const seconds = Number.parseFloat(raw);
  if (!Number.isNaN(seconds)) {
    return new Date(seconds * 1000).toLocaleString();
  }

  const asDate = new Date(raw);
  if (!Number.isNaN(asDate.getTime())) {
    return asDate.toLocaleString();
  }

  return raw;
}