import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { TriggerProps } from "@/ui/trigger";
import telegramIcon from "@/assets/icons/integrations/telegram.svg";

interface OnMentionConfiguration {
  chatId?: string;
}

interface OnMentionMetadata {
  chatId?: string;
  chatName?: string;
}

interface OnMentionEventData {
  message_id?: number;
  text?: string;
  date?: number;
  chat?: {
    id?: number;
    type?: string;
    title?: string;
  };
  from?: {
    id?: number;
    first_name?: string;
    username?: string;
  };
}

export const onMentionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnMentionEventData | undefined;
    const title = eventData?.text?.trim() ? eventData.text : "Bot mention";
    const subtitle = buildSubtitle(
      eventData?.from?.username ? `Mention by @${eventData.from.username}` : "Mention",
      context.event?.createdAt,
    );

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnMentionEventData | undefined;

    return {
      "Message ID": stringOrDash(eventData?.message_id),
      "Chat ID": stringOrDash(eventData?.chat?.id),
      "Chat Title": stringOrDash(eventData?.chat?.title),
      From: eventData?.from?.username ? `@${eventData.from.username}` : stringOrDash(eventData?.from?.first_name),
      Text: stringOrDash(eventData?.text),
    };
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnMentionConfiguration | undefined;
    const nodeMetadata = node.metadata as OnMentionMetadata | undefined;
    const metadataItems = [];

    const chatLabel = nodeMetadata?.chatName || configuration?.chatId;
    if (chatLabel) {
      metadataItems.push({ icon: "message-circle", label: chatLabel });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: telegramIcon,
      iconSlug: "telegram",
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnMentionEventData | undefined;
      const title = eventData?.text?.trim() ? eventData.text : "Bot mention";
      const subtitle = buildSubtitle(
        eventData?.from?.username ? `Mention by @${eventData.from.username}` : "Mention",
        lastEvent.createdAt,
      );

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
