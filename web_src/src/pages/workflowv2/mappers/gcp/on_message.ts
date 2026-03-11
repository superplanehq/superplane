import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";

export const onMessageTriggerRenderer: TriggerRenderer = {
  getEventState: (_context: TriggerEventContext) => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as Record<string, any> | undefined;
    const title = buildMessageTitle(data);

    const subtitleParts: string[] = [];
    const messageId = data?.messageId ? shortID(String(data.messageId)) : "";
    if (messageId) {
      subtitleParts.push(`#${messageId}`);
    }
    if (context.event?.createdAt) {
      subtitleParts.push(formatTimeAgo(new Date(context.event.createdAt)));
    }

    return { title, subtitle: subtitleParts.join(" · ") };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const data = context.event?.data as Record<string, any> | undefined;
    const details: Record<string, string> = {};
    if (data?.messageId) details["Message ID"] = String(data.messageId);
    if (data?.publishTime) details["Published At"] = new Date(data.publishTime as string).toLocaleString();
    if (context.event?.createdAt) details["Received At"] = new Date(context.event.createdAt).toLocaleString();
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { topicId?: string } | undefined;
    const topicId = configuration?.topicId;
    const eventTitleAndSubtitle = lastEvent
      ? onMessageTriggerRenderer.getTitleAndSubtitle({ event: lastEvent })
      : undefined;
    return {
      title: node.name || definition.label || "On Message",
      iconSrc: gcpPubSubIcon,
      iconSlug: definition.icon || "gcp",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata: topicId ? [{ icon: "message-square", label: topicId }] : [],
      ...(lastEvent && {
        lastEventData: {
          title: eventTitleAndSubtitle?.title ?? "Pub/Sub message",
          subtitle: eventTitleAndSubtitle?.subtitle ?? formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function buildMessageTitle(data?: Record<string, any>): string {
  const payload = data?.data;

  if (typeof payload === "string") {
    return truncate(payload, 72) || "Pub/Sub message";
  }

  if (payload && typeof payload === "object") {
    const record = payload as Record<string, unknown>;
    const labelCandidates = [record.title, record.name, record.message, record.action, record.type];
    const label = labelCandidates.find(
      (value): value is string => typeof value === "string" && value.trim().length > 0,
    );
    if (label) {
      return truncate(label, 72);
    }
  }

  const messageId = data?.messageId ? shortID(String(data.messageId)) : "";
  return messageId ? `Message ${messageId}` : "Pub/Sub message";
}

function shortID(value: string): string {
  return value.slice(0, 8);
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, maxLength)}...`;
}
