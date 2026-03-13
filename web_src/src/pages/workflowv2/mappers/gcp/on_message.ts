import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { formatTimeAgo } from "@/utils/date";
import gcpPubSubIcon from "@/assets/icons/integrations/gcp.pubsub.svg";

export const onMessageTriggerRenderer: TriggerRenderer = {
  getEventState: (_context: TriggerEventContext) => "triggered",

  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const data = context.event?.data as Record<string, any> | undefined;
    const messageId = data?.messageId ? shortID(String(data.messageId)) : "";
    const title = messageId ? `Received Pub/Sub message · ${messageId}` : "Received Pub/Sub message";

    const subtitleParts: string[] = [];
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
    const configuration = node.configuration as
      | { topic?: string; topicId?: string; subscription?: string; subscriptionId?: string }
      | undefined;
    const topic = configuration?.topic || configuration?.topicId;
    const subscription = configuration?.subscription || configuration?.subscriptionId;
    const metadata = [];
    if (topic) {
      metadata.push({ icon: "message-square", label: topic });
    }
    if (subscription) {
      metadata.push({ icon: "radio", label: subscription });
    }
    const eventTitleAndSubtitle = lastEvent
      ? onMessageTriggerRenderer.getTitleAndSubtitle({ event: lastEvent })
      : undefined;
    return {
      title: node.name || definition.label || "On Message",
      iconSrc: gcpPubSubIcon,
      iconSlug: definition.icon || "gcp",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass(definition.color ?? "gray"),
      metadata,
      ...(lastEvent && {
        lastEventData: {
          title: eventTitleAndSubtitle?.title ?? "Received Pub/Sub message",
          subtitle: eventTitleAndSubtitle?.subtitle ?? formatTimeAgo(new Date(lastEvent.createdAt)),
          receivedAt: new Date(lastEvent.createdAt),
          state: "triggered",
          eventId: lastEvent.id,
        },
      }),
    };
  },
};

function shortID(value: string): string {
  return value.slice(0, 8);
}
