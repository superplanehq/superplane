import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import type React from "react";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import type { TriggerProps } from "@/ui/trigger";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { stringOrDash } from "../utils";

interface OnAppMentionConfiguration {
  channel?: string;
}

interface OnAppMentionMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

interface AppMentionEventData {
  channel?: string;
  text?: string;
  user?: string;
  ts?: string;
  event_ts?: string;
  thread_ts?: string;
}

/**
 * Renderer for the "slack.onAppMention" trigger
 */
export const onAppMentionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as AppMentionEventData | undefined;

    return {
      title: mentionTitle(eventData),
      subtitle: buildSubtitle(mentionByline(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as AppMentionEventData | undefined;

    return {
      "Mentioned At": formatSlackTimestamp(eventData?.ts || eventData?.event_ts) || "",
      Channel: stringOrDash(eventData?.channel),
      User: stringOrDash(eventData?.user),
      Text: stringOrDash(eventData?.text),
      "Thread Timestamp": stringOrDash(eventData?.thread_ts),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnAppMentionMetadata | undefined;
    const configuration = node.configuration as OnAppMentionConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: mentionMetadataItems(metadata, configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onAppMentionTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

function mentionTitle(eventData?: AppMentionEventData): string {
  return eventData?.text?.trim() ? eventData.text : "App mention";
}

function mentionByline(eventData?: AppMentionEventData): string {
  return eventData?.user ? `Mention by ${eventData.user}` : "Mention";
}

function mentionMetadataItems(metadata?: OnAppMentionMetadata, configuration?: OnAppMentionConfiguration) {
  const channelLabel = metadata?.channel?.name || configuration?.channel;
  if (!channelLabel) {
    return [];
  }

  return [
    {
      icon: "hash",
      label: channelLabel,
    },
  ];
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
